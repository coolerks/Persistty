import xterm from '@xterm/headless';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { pathToFileURL } from 'node:url';
const { Terminal } = xterm;

export async function parse(frames, cols = 100, rows = 30, split = false) {
  const term = new Terminal({ cols, rows, scrollback: 5000, allowProposedApi: true });
  for (const frame of frames) {
    const bytes = Buffer.from(frame, 'base64');
    const chunks = split ? Array.from(bytes, byte => Uint8Array.of(byte)) : [new Uint8Array(bytes)];
    for (const chunk of chunks) await new Promise(resolve => term.write(chunk, resolve));
  }
  const buffer = b => Array.from({ length: b.length }, (_, i) => b.getLine(i).translateToString(true));
  const cells = b => Array.from({ length: b.length }, (_, y) => Array.from({length:cols}, (_, x) => {
    const cell = b.getLine(y).getCell(x);
    return [cell.getChars(),cell.getWidth(),cell.getFgColor(),cell.getBgColor(),cell.getFgColorMode(),cell.getBgColorMode(),cell.isBold(),cell.isItalic(),cell.isUnderline()];
  }));
  const result = { active: term.buffer.active.type, normal: buffer(term.buffer.normal),
    alternate: buffer(term.buffer.alternate), normal_cells: cells(term.buffer.normal),
    alternate_cells: cells(term.buffer.alternate), cursor: [term.buffer.active.cursorX, term.buffer.active.cursorY] };
  term.dispose();
  return result;
}

export async function analyze(data) {
  const summary = {};
  for (const [name, record] of Object.entries(data.records)) {
    const baseline = await parse(record.frames, record.cols, record.rows);
    const fragmented = await parse(record.frames, record.cols, record.rows, true);
    const bytes = Buffer.concat(record.frames.map(f => Buffer.from(f, 'base64')));
    const lines = [...baseline.normal, ...baseline.alternate];
    summary[name] = { bytes: bytes.length, sha256: createHash('sha256').update(bytes).digest('hex'),
      one_byte_chunks_equivalent: JSON.stringify(baseline) === JSON.stringify(fragmented),
      active_buffer: baseline.active, normal_lines: baseline.normal.length,
      has_history_first: lines.some(l => l.includes('HIST_0001')),
      has_gap_first: lines.some(l => l.includes('GAP_0001')),
      has_gap_last: lines.some(l => l.includes('GAP_0080')),
      has_tui: lines.some(l => l.includes('CURSES_D07')),
      has_dimensions: lines.some(l => l.includes(`SIZE_${record.cols}x${record.rows}`)),
      has_unicode: lines.some(l => l.includes('雪')),
      has_exit: lines.some(l => l.includes('TUI_EXIT_D07')),
      wide_cells: [...baseline.normal_cells.flat(),...baseline.alternate_cells.flat()].filter(c => c[1]===2).length,
      colored_cells: [...baseline.normal_cells.flat(),...baseline.alternate_cells.flat()].filter(c => c[4]!==0).length,
      resize_ack: record.resize_ack, attach_reaped: !record.stats.active && record.stats.started === record.stats.reaped };
  }
  const capture = data.capture;
  const combined = await parse([Buffer.from(capture).toString('base64'), ...data.records.gap.frames]);
  const combinedLines = [...combined.normal, ...combined.alternate];
  summary.capture_plus_attach = { active_buffer: combined.active,
    has_history_first: combinedLines.some(l => l.includes('HIST_0001')),
    active_has_history_first: combined[combined.active].some(l => l.includes('HIST_0001')),
    missing_gap_first: !combinedLines.some(l => l.includes('GAP_0001')),
    capture_has_history_first: capture.includes('HIST_0001'),
    capture_lines: capture.split('\n').length };
  return summary;
}

export function checks(summary) {
  return {
    bytewise_cells_equivalent: Object.values(summary).filter(s => 'bytes' in s).every(s => s.one_byte_chunks_equivalent),
    every_attach_reaped: Object.values(summary).filter(s => 'bytes' in s).every(s => s.attach_reaped && s.resize_ack),
    raw_attach_lacks_full_history: !summary.plain.has_history_first,
    capture_has_ordinary_history: summary.capture_plus_attach.capture_has_history_first,
    capture_attach_gap_counterexample: summary.capture_plus_attach.missing_gap_first && summary.gap.has_gap_last,
    outer_alternate_hides_captured_history: summary.capture_plus_attach.active_buffer==='alternate' && !summary.capture_plus_attach.active_has_history_first,
    curses_redraw_and_reconnect: ['tui100','tui80','tui120','tui_reconnect'].every(n => summary[n].has_tui && summary[n].has_dimensions && summary[n].has_unicode && summary[n].wide_cells>0 && summary[n].colored_cells>0),
    curses_exit: summary.exit.has_exit && !summary.exit.has_tui
  };
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  const analysis = await analyze(JSON.parse(readFileSync(process.argv[2], 'utf8')));
  console.log(JSON.stringify({analysis,checks:checks(analysis)}));
}
