import xterm from '@xterm/headless';
import serializer from '@xterm/addon-serialize';
import { randomUUID } from 'node:crypto';

const { Terminal } = xterm;
const { SerializeAddon } = serializer;
const SCROLLBACK = 200;
const MAX_EVENT = 64 * 1024;
const MAX_SNAPSHOT = 1024 * 1024;
const MAX_PENDING = 64;
const MAX_PENDING_BYTES = 2 * 1024 * 1024;

function enqueue(target, action, weight = 0) {
  if (target.pending >= MAX_PENDING || target.pendingBytes + weight > MAX_PENDING_BYTES) {
    throw new Error('queue_full');
  }
  target.pending++;
  target.pendingBytes += weight;
  const result = target.queue.then(action).finally(() => {
    target.pending--;
    target.pendingBytes -= weight;
  });
  target.queue = result.catch(() => {});
  return result;
}

function size(cols, rows) {
  if (!Number.isInteger(cols) || !Number.isInteger(rows) || cols < 1 || rows < 1 || cols > 200 || rows > 100) {
    throw new Error('invalid_size');
  }
}

export function terminal(cols = 40, rows = 8) {
  size(cols, rows);
  return new Terminal({ cols, rows, scrollback: SCROLLBACK, allowProposedApi: true });
}

export function write(term, bytes) {
  return new Promise(resolve => term.write(bytes, resolve));
}

export function state(term) {
  const buffer = b => ({ cursor: [b.cursorX, b.cursorY], baseY: b.baseY,
    lines: Array.from({ length: b.length }, (_, y) => {
      const line = b.getLine(y);
      return { wrapped: line.isWrapped, cells: Array.from({ length: term.cols }, (_, x) => {
        const cell = line.getCell(x);
        return [cell.getChars(), cell.getWidth(), cell.getFgColor(), cell.getBgColor(),
          cell.getFgColorMode(), cell.getBgColorMode(), cell.isBold(), cell.isItalic(),
          cell.isUnderline(), cell.isBlink(), cell.isInverse(), cell.isInvisible(),
          cell.isDim(), cell.isStrikethrough(), cell.isOverline()];
      }) };
    }) });
  return { cols: term.cols, rows: term.rows, active: term.buffer.active.type,
    normal: buffer(term.buffer.normal), alternate: buffer(term.buffer.alternate), modes: term.modes };
}

export class Owner {
  constructor({ cols = 40, rows = 8, epoch = randomUUID(), eventCap = 64, byteCap = 256 * 1024 } = {}) {
    if (typeof epoch !== 'string' || !epoch || epoch.length > 128 || !Number.isSafeInteger(eventCap) ||
      eventCap < 1 || eventCap > 4096 || !Number.isSafeInteger(byteCap) || byteCap < 1 || byteCap > 1024 * 1024) {
      throw new Error('invalid_owner');
    }
    this.term = terminal(cols, rows);
    this.addon = new SerializeAddon();
    this.term.loadAddon(this.addon);
    this.epoch = epoch;
    this.seq = 0;
    this.ring = [];
    this.bytes = 0;
    this.dropped = 0;
    this.eventCap = eventCap;
    this.byteCap = byteCap;
    this.queue = Promise.resolve();
    this.pending = 0;
    this.pendingBytes = 0;
  }

  enqueue(action, weight = 0) {
    return enqueue(this, action, weight);
  }

  apply(input) {
    let payload;
    if (input?.kind === 'bytes' && input.bytes instanceof Uint8Array) {
      if (input.bytes.length > MAX_EVENT || input.bytes.length > this.byteCap) throw new Error('event_too_large');
      payload = { kind: 'bytes', bytes: Uint8Array.from(input.bytes) };
    } else if (input?.kind === 'resize') {
      size(input.cols, input.rows);
      payload = { kind: 'resize', cols: input.cols, rows: input.rows };
    } else throw new Error('invalid_event');
    return this.enqueue(async () => {
      if (!Number.isSafeInteger(this.seq + 1)) throw new Error('sequence_exhausted');
      if (payload.kind === 'bytes') await write(this.term, payload.bytes);
      else this.term.resize(payload.cols, payload.rows);
      const event = { epoch: this.epoch, seq: ++this.seq, ...payload };
      this.ring.push(event);
      this.bytes += payload.bytes?.length ?? 0;
      while (this.ring.length > this.eventCap || this.bytes > this.byteCap) {
        this.bytes -= this.ring.shift().bytes?.length ?? 0;
        this.dropped++;
      }
      return cloneEvent(event);
    }, payload.bytes?.length ?? 0);
  }

  snapshot() {
    return this.enqueue(() => {
      const serialized = this.addon.serialize({ scrollback: SCROLLBACK, excludeModes: false, excludeAltBuffer: false });
      if (Buffer.byteLength(serialized) > MAX_SNAPSHOT) throw new Error('snapshot_too_large');
      return { epoch: this.epoch, through_seq: this.seq, cols: this.term.cols, rows: this.term.rows, serialized };
    });
  }

  tail(epoch, after) {
    return this.enqueue(() => {
      if (epoch !== this.epoch) return { status: 'epoch_mismatch' };
      if (!Number.isSafeInteger(after) || after < 0 || after > this.seq) return { status: 'invalid_cursor' };
      const oldest = this.ring[0]?.seq ?? this.seq + 1;
      if (after + 1 < oldest) return { status: 'resync_required', oldest, newest: this.seq, dropped: this.dropped };
      return { status: 'ok', oldest, newest: this.seq, dropped: this.dropped,
        events: this.ring.filter(event => event.seq > after).map(cloneEvent) };
    });
  }

  dispose() { this.term.dispose(); }
}

function cloneEvent(event) {
  return event.kind === 'bytes' ? { ...event, bytes: Uint8Array.from(event.bytes) } : { ...event };
}

export class Observer {
  constructor(epoch) {
    this.epoch = epoch;
    this.term = null;
    this.through = -1;
    this.stopped = true;
    this.writeCount = 0;
    this.resizeCount = 0;
    this.titles = [];
    this.queue = Promise.resolve();
    this.pending = 0;
    this.pendingBytes = 0;
  }

  enqueue(action, weight = 0) {
    return enqueue(this, action, weight);
  }

  restore(snapshot) {
    const copy = { ...snapshot };
    const weight = typeof copy.serialized === 'string' ? Buffer.byteLength(copy.serialized) : 0;
    if (weight > MAX_SNAPSHOT) throw new Error('invalid_snapshot');
    return this.enqueue(async () => {
      if (copy.epoch !== this.epoch) return 'epoch_mismatch';
      if (!Number.isSafeInteger(copy.through_seq) || copy.through_seq < 0 ||
        typeof copy.serialized !== 'string') {
        throw new Error('invalid_snapshot');
      }
      if (copy.through_seq < this.through) return 'stale_snapshot';
      const fresh = terminal(copy.cols, copy.rows);
      fresh.onTitleChange(title => this.titles.push(title));
      await write(fresh, copy.serialized);
      this.term?.dispose();
      this.term = fresh;
      this.through = copy.through_seq;
      this.stopped = false;
      return 'restored';
    }, weight);
  }

  accept(event) {
    if (event?.kind === 'bytes' && event.bytes instanceof Uint8Array && event.bytes.length > MAX_EVENT) {
      throw new Error('event_too_large');
    }
    // Copy before queueing: caller mutation cannot alter an accepted event.
    const copy = event?.kind === 'bytes' && event.bytes instanceof Uint8Array ? cloneEvent(event) : { ...event };
    return this.enqueue(async () => {
      if (copy.epoch !== this.epoch) return 'epoch_mismatch';
      if (this.stopped) return 'resync_required';
      if (!Number.isSafeInteger(copy.seq) || copy.seq < 1) throw new Error('invalid_sequence');
      if (copy.seq <= this.through) return 'duplicate';
      if (copy.seq !== this.through + 1) { this.stopped = true; return 'gap'; }
      if (copy.kind === 'bytes' && copy.bytes instanceof Uint8Array && copy.bytes.length <= MAX_EVENT) {
        await write(this.term, copy.bytes);
        this.writeCount++;
      } else if (copy.kind === 'resize') {
        size(copy.cols, copy.rows);
        this.term.resize(copy.cols, copy.rows);
        this.resizeCount++;
      } else throw new Error('invalid_event');
      this.through = copy.seq;
      return 'applied';
    }, copy.bytes instanceof Uint8Array ? copy.bytes.length : 0);
  }

  dispose() { this.term?.dispose(); }
}

export const bytes = text => ({ kind: 'bytes', bytes: Buffer.from(text) });

export async function compare(prefix, suffix, options = {}) {
  const owner = new Owner(options);
  const observer = new Observer(owner.epoch);
  const titles = [];
  owner.term.onTitleChange(title => titles.push(title));
  try {
    for (const event of prefix) await owner.apply(event);
    const snapshot = await owner.snapshot();
    const titleOffset = titles.length;
    await observer.restore(snapshot);
    for (const event of suffix) await observer.accept(await owner.apply(event));
    const original = state(owner.term);
    const restored = state(observer.term);
    const sameState = JSON.stringify(original) === JSON.stringify(restored);
    const sameTitles = JSON.stringify(titles.slice(titleOffset)) === JSON.stringify(observer.titles);
    const differingFields = [];
    for (const key of ['cols', 'rows', 'active', 'modes']) {
      if (JSON.stringify(original[key]) !== JSON.stringify(restored[key])) differingFields.push(key);
    }
    for (const buffer of ['normal', 'alternate']) {
      for (const key of ['cursor', 'baseY']) {
        if (JSON.stringify(original[buffer][key]) !== JSON.stringify(restored[buffer][key])) differingFields.push(`${buffer}.${key}`);
      }
      if (original[buffer].lines.length !== restored[buffer].lines.length) differingFields.push(`${buffer}.line_count`);
      const wrap = value => value[buffer].lines.map(line => line.wrapped);
      if (JSON.stringify(wrap(original)) !== JSON.stringify(wrap(restored))) differingFields.push(`${buffer}.wrapped`);
      const cells = value => value[buffer].lines.map(line => line.cells);
      if (JSON.stringify(cells(original)) !== JSON.stringify(cells(restored))) differingFields.push(`${buffer}.cells`);
    }
    if (!sameTitles) differingFields.push('title_events');
    return { equivalent: sameState && sameTitles, same_state: sameState, same_title_events: sameTitles,
      differing_fields: differingFields,
      original, restored, original_titles: titles.slice(titleOffset), restored_titles: observer.titles,
      through_seq: snapshot.through_seq };
  } finally { owner.dispose(); observer.dispose(); }
}
