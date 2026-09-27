import { createHash } from 'node:crypto';
import { openSync, readSync, fstatSync, closeSync, constants } from 'node:fs';
import { pathToFileURL } from 'node:url';
import { compare } from './model.mjs';
import { scenarioSummary } from './scenarios.mjs';

const hash = data => createHash('sha256').update(data).digest('hex');

export function validateRecord(data) {
  // One connection only. Other records from the shared probe are never concatenated.
  const record = data?.records?.tui120;
  if (!record || !Number.isInteger(record.cols) || !Number.isInteger(record.rows) ||
    record.cols < 1 || record.cols > 200 || record.rows < 1 || record.rows > 100 ||
    !Array.isArray(record.frames) || !record.frames.length || record.frames.length > 4096) {
    throw new Error('invalid_record');
  }
  let total = 0;
  const frames = record.frames.map(frame => {
    if (typeof frame !== 'string' || frame.length > 87384) throw new Error('invalid_frame');
    const bytes = Buffer.from(frame, 'base64');
    if (bytes.toString('base64') !== frame || !bytes.length || bytes.length > 65536) throw new Error('invalid_frame');
    total += bytes.length;
    if (total > 256 * 1024) throw new Error('record_too_large');
    return bytes;
  });
  return { frames, cols: record.cols, rows: record.rows, total };
}

function summary(comparison, description) {
  return { ...description, equivalent: comparison.equivalent, same_state: comparison.same_state,
    same_title_events: comparison.same_title_events,
    differing_fields: comparison.differing_fields,
    continuous_state_sha256: hash(JSON.stringify(comparison.original)),
    restored_state_sha256: hash(JSON.stringify(comparison.restored)) };
}

export async function analyze(data) {
  const record = validateRecord(data);
  const events = record.frames.map(bytes => ({ kind: 'bytes', bytes }));
  const boundaries = [...new Set([0, 1, Math.floor(events.length / 2), events.length - 1, events.length])];
  const cuts = [];
  for (const index of boundaries) {
    cuts.push(summary(await compare(events.slice(0, index), events.slice(index), record),
      { kind: 'frame_boundary', after_frames: index }));
  }
  // Select a bounded set of offsets without guessing ANSI parser state or filtering bytes.
  const largest = record.frames.reduce((best, frame, index) => frame.length > record.frames[best].length ? index : best, 0);
  const frame = record.frames[largest];
  const offsets = [...new Set([1, 2, Math.floor(frame.length / 2), frame.length - 1])].filter(n => n > 0 && n < frame.length);
  for (const offset of offsets) {
    const prefix = [...events.slice(0, largest), { kind: 'bytes', bytes: frame.subarray(0, offset) }];
    const suffix = [{ kind: 'bytes', bytes: frame.subarray(offset) }, ...events.slice(largest + 1)];
    cuts.push(summary(await compare(prefix, suffix, record), { kind: 'artificial_byte_split', frame: largest, offset }));
  }
  const synthetic = await scenarioSummary();
  const positives = Object.values(synthetic).filter(s => s.expected_equivalence === true);
  const negatives = Object.values(synthetic).filter(s => s.expected_equivalence === false);
  return { analysis: { versions: { headless: '5.5.0', serialize: '0.13.0', xterm_peer: '5.5.0' },
    synthetic, real_single_connection: { record: 'tui120', frames: record.frames.length, bytes: record.total,
      sha256: hash(Buffer.concat(record.frames)), cols: record.cols, rows: record.rows, cuts,
      equivalent_cuts: cuts.filter(c => c.equivalent).length, divergent_cuts: cuts.filter(c => !c.equivalent).length,
      local_sequence_only: true, raw_saved: false } },
    checks: { public_api_smoke: synthetic.smoke.equivalent,
      fixed_complete_basics_equivalent: positives.every(s => s.equivalent),
      pending_counterexamples_observed: negatives.every(s => !s.equivalent),
      osc_requires_event_comparison: synthetic.osc_bel_pending.same_state && !synthetic.osc_bel_pending.same_title_events,
      complete_region_and_charset_not_restored: !synthetic.scroll_region_completed.equivalent && !synthetic.charset_completed.equivalent,
      real_single_connection_compared: cuts.length > 0 } };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const fd = openSync(process.argv[2], constants.O_RDONLY | constants.O_NOFOLLOW | constants.O_NONBLOCK);
  try {
    const info = fstatSync(fd);
    if (!info.isFile() || info.size > 2 * 1024 * 1024 || (info.mode & 0o777) !== 0o600 || info.uid !== process.getuid()) {
      throw new Error('unsafe_input_file');
    }
    const raw = Buffer.alloc(2 * 1024 * 1024 + 1);
    let count = 0;
    while (count < raw.length) {
      const read = readSync(fd, raw, count, raw.length - count, null);
      if (!read) break;
      count += read;
    }
    if (count === raw.length) throw new Error('input_too_large');
    console.log(JSON.stringify(await analyze(JSON.parse(raw.subarray(0, count).toString('utf8')))));
  } finally { closeSync(fd); }
}
