import test from 'node:test';
import assert from 'node:assert/strict';
import { parse } from './analyze.mjs';

test('UTF-8, CSI and OSC split bytewise preserve xterm state', async () => {
  const frames = [Buffer.from('\x1b[31m雪e\u0301\x1b[0m\r\n\x1b]0;synthetic\x07END').toString('base64')];
  assert.deepEqual(await parse(frames), await parse(frames, 100, 30, true));
});
test('alternate enter and leave is interpreted by the library', async () => {
  const frames = [Buffer.from('NORMAL\x1b[?1049h\x1b[H\x1b[2JALT\x1b[?1049l').toString('base64')];
  const result = await parse(frames);
  assert.equal(result.active, 'normal');
  assert.ok(result.normal.some(l => l.includes('NORMAL')));
  assert.deepEqual(result, await parse(frames, 100, 30, true));
});
