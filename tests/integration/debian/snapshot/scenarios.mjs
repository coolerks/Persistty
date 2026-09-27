import { bytes, compare } from './model.mjs';

export function cases() {
  const unicode = Buffer.from('雪');
  return [
    ['smoke', [bytes('Hello')], [bytes(' world')], true],
    ['history_unicode_sgr_wrapped', [bytes(Array.from({ length: 15 }, (_, i) => `line_${i} 雪 e\u0301\r\n`).join('')),
      bytes('\x1b[1;3;4;31;44mcolored\x1b[0m'), bytes('a'.repeat(90))], [bytes('\r\ntail')], true],
    ['alternate_exit_resize', [bytes('normal\r\n'), bytes('\x1b[?1049h\x1b[2J\x1b[H\x1b[32mALT 雪\x1b[0m')],
      [bytes('\x1b[?1049l'), { kind: 'resize', cols: 50, rows: 10 }, bytes('\r\nexit')], true],
    ['resize_completed', [bytes('normal 雪'), { kind: 'resize', cols: 50, rows: 10 }], [bytes('\r\nafter')], true],
    ['modes_completed', [bytes('\x1b[?1h\x1b[?2004h\x1b[4h\x1b[?7lM')], [bytes('N')], true],
    ['utf8_one_byte_pending', [{ kind: 'bytes', bytes: unicode.subarray(0, 1) }],
      [{ kind: 'bytes', bytes: unicode.subarray(1) }], false],
    ['utf8_two_bytes_pending', [{ kind: 'bytes', bytes: unicode.subarray(0, 2) }],
      [{ kind: 'bytes', bytes: unicode.subarray(2) }], false],
    ['csi_parameters_pending', [bytes('\x1b[31')], [bytes('mX')], false],
    ['esc_pending', [bytes('\x1b')], [bytes('[31mX')], false],
    ['osc_bel_pending', [bytes('\x1b]0;SAFE_D08')], [bytes('\x07X')], false],
    ['osc_st_pending', [bytes('\x1b]0;SAFE_D08')], [bytes('\x1b\\X')], false],
    ['osc_st_esc_pending', [bytes('\x1b]0;SAFE_D08\x1b')], [bytes('\\X')], false],
    ['scroll_region_completed', [bytes('\x1b[2;6r\x1b[6;1Hregion')], [bytes('\r\nNEXT')], null],
    ['charset_completed', [bytes('\x1b(0')], [bytes('q')], null]
  ];
}

export async function scenarioSummary() {
  const result = {};
  for (const [name, prefix, suffix, expected] of cases()) {
    const comparison = await compare(prefix, suffix);
    result[name] = { expected_equivalence: expected, equivalent: comparison.equivalent,
      same_state: comparison.same_state, same_title_events: comparison.same_title_events,
      differing_fields: comparison.differing_fields,
      snapshot_through_seq: comparison.through_seq };
  }
  return result;
}
