import test from 'node:test';
import assert from 'node:assert/strict';
import { Owner, Observer, bytes, state } from './model.mjs';
import { scenarioSummary } from './scenarios.mjs';

test('public serialize compatibility and fixed-boundary/pending matrix', { timeout: 20000 }, async () => {
  const summary = await scenarioSummary();
  assert.equal(summary.smoke.equivalent, true);
  for (const value of Object.values(summary)) {
    if (value.expected_equivalence === true) assert.equal(value.equivalent, true);
    if (value.expected_equivalence === false) assert.equal(value.equivalent, false);
  }
  assert.equal(summary.osc_bel_pending.same_state, true);
  assert.equal(summary.osc_bel_pending.same_title_events, false);
});

test('queued snapshot includes only prior applied events and bytes are copied', async () => {
  const owner = new Owner();
  try {
    const input = bytes('first');
    const first = owner.apply(input);
    const snapshot = owner.snapshot();
    const second = owner.apply(bytes('second'));
    input.bytes.fill(88);
    assert.equal((await first).seq, 1);
    const saved = await snapshot;
    assert.equal(saved.through_seq, 1);
    assert.equal((await second).seq, 2);
    const observer = new Observer(owner.epoch);
    try {
      await observer.restore(saved);
      for (const event of (await owner.tail(owner.epoch, saved.through_seq)).events) await observer.accept(event);
      assert.deepEqual(state(observer.term), state(owner.term));
      assert.equal(state(owner.term).normal.lines[0].cells.slice(0, 5).map(c => c[0]).join(''), 'first');
    } finally { observer.dispose(); }
  } finally { owner.dispose(); }
});

test('duplicate epoch and gaps are handled before write or resize; gap stays stopped', async () => {
  const owner = new Owner({ epoch: 'one' });
  const observer = new Observer('one');
  try {
    const empty = await owner.snapshot();
    await observer.restore(empty);
    const first = await owner.apply(bytes('A'));
    assert.equal(await observer.accept(first), 'applied');
    assert.equal(await observer.accept(first), 'duplicate');
    assert.equal(await observer.accept({ ...first, epoch: 'old', seq: 2 }), 'epoch_mismatch');
    assert.equal(observer.writeCount, 1);
    const resize = await owner.apply({ kind: 'resize', cols: 50, rows: 10 });
    assert.equal(await observer.accept(resize), 'applied');
    assert.equal(await observer.accept(resize), 'duplicate');
    assert.equal(observer.resizeCount, 1);
    assert.equal(await observer.restore(empty), 'stale_snapshot');
    const missing = await owner.apply(bytes('B'));
    const later = await owner.apply(bytes('C'));
    assert.equal(await observer.accept(later), 'gap');
    assert.equal(await observer.accept(missing), 'resync_required');
    assert.equal(observer.writeCount, 1);
    await observer.restore(await owner.snapshot());
    assert.deepEqual(state(observer.term), state(owner.term));
    const nextEpoch = new Owner({ epoch: 'two' });
    try {
      assert.equal((await nextEpoch.apply(bytes('N'))).seq, 1);
      assert.equal(await observer.restore(await nextEpoch.snapshot()), 'epoch_mismatch');
    } finally { nextEpoch.dispose(); }
  } finally { observer.dispose(); owner.dispose(); }
});

test('event cap deterministically evicts tail after queued snapshot; fresh resync matches', async () => {
  const owner = new Owner({ eventCap: 2 });
  const observer = new Observer(owner.epoch);
  try {
    const snapshotPromise = owner.snapshot();
    const writes = ['A', 'B', 'C'].map(text => owner.apply(bytes(text)));
    const snapshot = await snapshotPromise;
    await Promise.all(writes);
    const tail = await owner.tail(owner.epoch, snapshot.through_seq);
    assert.equal(tail.status, 'resync_required');
    assert.equal(tail.oldest, 2);
    assert.equal(owner.ring.length, 2);
    await observer.restore(await owner.snapshot());
    await observer.accept(await owner.apply(bytes('D')));
    assert.deepEqual(state(observer.term), state(owner.term));
  } finally { observer.dispose(); owner.dispose(); }
});

test('byte cap and single-event rejection are independent and epoch/cursor checked', async () => {
  const owner = new Owner({ eventCap: 8, byteCap: 5 });
  try {
    await owner.apply(bytes('abc'));
    await owner.apply(bytes('def'));
    assert.equal(owner.bytes, 3);
    assert.equal(owner.ring.length, 1);
    assert.equal((await owner.tail(owner.epoch, 0)).status, 'resync_required');
    assert.equal((await owner.tail('old', 1)).status, 'epoch_mismatch');
    assert.equal((await owner.tail(owner.epoch, 3)).status, 'invalid_cursor');
    assert.throws(() => owner.apply(bytes('123456')), /event_too_large/);
    assert.equal(owner.seq, 2);
    assert.equal((await owner.apply(bytes('x'))).seq, 3);
  } finally { owner.dispose(); }
});

test('tail output cannot mutate owner retained bytes', async () => {
  const owner = new Owner();
  try {
    await owner.apply(bytes('ABC'));
    const first = await owner.tail(owner.epoch, 0);
    first.events[0].bytes.fill(88);
    assert.equal(Buffer.from((await owner.tail(owner.epoch, 0)).events[0].bytes).toString(), 'ABC');
  } finally { owner.dispose(); }
});

test('pending event and byte budgets reject before application and release on completion', async () => {
  const owner = new Owner();
  const observer = new Observer(owner.epoch);
  try {
    const pending = Array.from({ length: 64 }, () => owner.snapshot());
    assert.throws(() => owner.apply(bytes('rejected')), /queue_full/);
    assert.equal(owner.seq, 0);
    await Promise.all(pending);
    assert.equal(owner.pending, 0);
    const writes = Array.from({ length: 32 }, () => owner.apply({ kind: 'bytes', bytes: Buffer.alloc(65536) }));
    assert.throws(() => owner.apply(bytes('over byte budget')), /queue_full/);
    assert.equal(owner.pendingBytes, 2 * 1024 * 1024);
    await Promise.all(writes);
    assert.equal(owner.pendingBytes, 0);
    await observer.restore(await owner.snapshot());
    const accepts = Array.from({ length: 64 }, () => observer.accept({ epoch: owner.epoch, seq: 1, kind: 'resize', cols: 40, rows: 8 }));
    assert.throws(() => observer.restore({ epoch: owner.epoch }), /queue_full/);
    await Promise.all(accepts);
    assert.equal(observer.pending, 0);
    const saved = { epoch: owner.epoch, through_seq: owner.seq, cols: 40, rows: 8, serialized: '\x00'.repeat(1024 * 1024) };
    const restores = [observer.restore(saved), observer.restore(saved)];
    assert.throws(() => observer.restore(saved), /queue_full/);
    assert.equal(observer.pendingBytes, 2 * 1024 * 1024);
    await Promise.all(restores);
    assert.equal(observer.pendingBytes, 0);
    assert.throws(() => observer.accept({ kind: 'bytes', bytes: Buffer.alloc(65537) }), /event_too_large/);
    assert.equal(observer.pending, 0);
  } finally { observer.dispose(); owner.dispose(); }
});

test('queued restore freezes snapshot and failed actions release queue reservations', async () => {
  const owner = new Owner();
  const observer = new Observer(owner.epoch);
  try {
    await owner.apply(bytes('immutable'));
    const saved = await owner.snapshot();
    const restore = observer.restore(saved);
    saved.serialized = 'mutated';
    saved.through_seq = 100;
    saved.cols = 1;
    assert.equal(await restore, 'restored');
    assert.deepEqual(state(observer.term), state(owner.term));
    assert.equal(observer.through, 1);
    await assert.rejects(observer.restore({ epoch: owner.epoch, through_seq: 2, serialized: 'x', cols: 0, rows: 8 }), /invalid_size/);
    assert.equal(observer.pending, 0);
    assert.equal(observer.pendingBytes, 0);
    assert.equal(await observer.accept(await owner.apply(bytes(' tail'))), 'applied');
  } finally { observer.dispose(); owner.dispose(); }
});
