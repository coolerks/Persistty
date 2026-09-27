import test from 'node:test';
import assert from 'node:assert/strict';
import { analyze, validateRecord } from './analyze.mjs';
import { mkdtempSync, writeFileSync, chmodSync, symlinkSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { spawnSync } from 'node:child_process';

test('record validation bounds frames sizes and canonical base64', () => {
  const record = frames => ({ records: { tui120: { cols: 80, rows: 24, frames } } });
  for (const frames of [[], ['?'], ['AA'], [''], [Buffer.alloc(65537).toString('base64')], Array(4097).fill('WA==')]) {
    assert.throws(() => validateRecord(record(frames)));
  }
  assert.throws(() => validateRecord(record(Array(5).fill(Buffer.alloc(65536).toString('base64')))));
  assert.equal(validateRecord(record(['WA=='])).total, 1);
  for (const [cols, rows] of [[0, 24], [201, 24], [80, 101], [1.5, 24]]) {
    assert.throws(() => validateRecord({ records: { tui120: { cols, rows, frames: ['WA=='] } } }));
  }
});

test('CLI rejects unsafe and oversized files before analysis', () => {
  const directory = mkdtempSync(join(tmpdir(), 'persistty-snapshot-test-'));
  const path = join(directory, 'input.json');
  const link = join(directory, 'link.json');
  const run = file => spawnSync(process.execPath, [fileURLToPath(new URL('./analyze.mjs', import.meta.url)), file],
    { timeout: 5000, maxBuffer: 65536 });
  try {
    writeFileSync(path, '{}', { mode: 0o600 });
    symlinkSync(path, link);
    assert.notEqual(run(link).status, 0);
    chmodSync(path, 0o644);
    assert.notEqual(run(path).status, 0);
    chmodSync(path, 0o600);
    writeFileSync(path, Buffer.alloc(2 * 1024 * 1024 + 1));
    assert.notEqual(run(path).status, 0);
    const output = run(directory);
    assert.notEqual(output.status, 0);
    assert.equal(output.stdout.length, 0);
  } finally { rmSync(directory, { recursive: true, force: true }); }
});

test('analysis selects exactly one connection and retains positive/negative observations', { timeout: 20000 }, async () => {
  const result = await analyze({ records: {
    tui120: { cols: 80, rows: 24, frames: [Buffer.from('safe 雪\x1b[31mX\x1b[0m').toString('base64')] },
    foreign: { frames: ['INVALID'] }
  } });
  assert.equal(result.analysis.real_single_connection.record, 'tui120');
  assert.equal(result.analysis.real_single_connection.frames, 1);
  assert.ok(result.analysis.real_single_connection.cuts.length >= 3);
  assert.ok(Object.values(result.checks).every(Boolean));
  assert.equal(JSON.stringify(result).includes('"cells"'), false);
  assert.equal(JSON.stringify(result).includes('"serialized"'), false);
});
