import { openSync, readSync, fstatSync, closeSync, constants } from 'node:fs';
import { createRequire } from 'node:module';

const require = createRequire(new URL('../history/package.json', import.meta.url));
const { Terminal } = require('@xterm/headless');

async function render(encoded) {
  if (typeof encoded !== 'string' || encoded.length > 350000) throw new Error('invalid_record');
  const bytes = Buffer.from(encoded, 'base64');
  if (!bytes.length || bytes.length > 256 * 1024 || bytes.toString('base64') !== encoded) {
    throw new Error('invalid_record');
  }
  const term = new Terminal({ cols: 100, rows: 30, scrollback: 0, allowProposedApi: true });
  await new Promise(resolve => term.write(new Uint8Array(bytes), resolve));
  const active = term.buffer.active.type;
  const visible = Array.from({ length: term.buffer.active.length }, (_, index) =>
    term.buffer.active.getLine(index).translateToString(true));
  const normalLength = term.buffer.normal.length;
  term.dispose();
  return { active, visible, normalLength };
}

const fd = openSync(process.argv[2], constants.O_RDONLY | constants.O_NOFOLLOW | constants.O_NONBLOCK);
try {
  const info = fstatSync(fd);
  if (!info.isFile() || info.size > 1024 * 1024 || (info.mode & 0o777) !== 0o600 ||
      info.uid !== process.getuid()) throw new Error('unsafe_input_file');
  const raw = Buffer.alloc(info.size + 1);
  let count = 0;
  while (count < raw.length) {
    const read = readSync(fd, raw, count, raw.length - count, null);
    if (!read) break;
    count += read;
  }
  if (count !== info.size) throw new Error('invalid_input_size');
  const records = JSON.parse(raw.subarray(0, count).toString('utf8'));
  if (Object.keys(records).sort().join(',') !== 'osc,stream,tui,utf8') throw new Error('invalid_records');
  const stream = await render(records.stream);
  const utf8 = await render(records.utf8);
  const osc = await render(records.osc);
  const tui = await render(records.tui);
  const streamNumbers = stream.visible.flatMap(line => Array.from(line.matchAll(/SEQ_(\d{4})/g), match => Number(match[1])));
  const checks = {
    stream_live_view_current: stream.active === 'alternate' && streamNumbers.at(-1) === 400 &&
                              streamNumbers.every((number, index) => index === 0 || number === streamNumbers[index - 1] + 1),
    stream_history_view_disjoint: streamNumbers[0] === 372,
    utf8_cell_restored: utf8.visible.some(line => line.includes('雪 UTF8_DONE')),
    osc_following_text_restored: osc.visible.some(line => line.includes('OSC_DONE')),
    tui_active_alternate_screen: tui.active === 'alternate',
    tui_cells_restored: tui.visible.some(line => line.includes('CURSES_W03')) &&
                        tui.visible.some(line => line.includes('ACTIVE_ALT_SCREEN')),
    live_view_has_no_duplicate_scrollback: utf8.normalLength === 30 && osc.normalLength === 30,
  };
  console.log(JSON.stringify({ checks, summary: {
    headless_version: '5.5.0', stream_buffer: stream.active,
    stream_visible_bounds: [streamNumbers[0] ?? 0, streamNumbers.at(-1) ?? 0],
    utf8_buffer: utf8.active,
    osc_buffer: osc.active, tui_buffer: tui.active,
    utf8_normal_length: utf8.normalLength, osc_normal_length: osc.normalLength,
  } }));
} finally {
  closeSync(fd);
}
