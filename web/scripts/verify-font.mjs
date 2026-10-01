import { URL } from "node:url";
import { Buffer } from "node:buffer";
import { readFileSync } from "node:fs";
import { createHash } from "node:crypto";
const meta = JSON.parse(readFileSync(new URL("../licenses/jetbrains-mono-nerd-font.json", import.meta.url), "utf8"));
const data = readFileSync(new URL(`../public/fonts/${meta.file}`, import.meta.url));
if (createHash("sha256").update(data).digest("hex") !== meta.sha256) throw new Error("字体 SHA256 不匹配");
const tables = new Map();
for (let i = 0; i < data.readUInt16BE(4); i++) { const p = 12 + i * 16; tables.set(data.toString("ascii", p, p + 4), data.readUInt32BE(p + 8)); }
const name = tables.get("name"), cmap = tables.get("cmap"), hhea = tables.get("hhea"), hmtx = tables.get("hmtx");
if ([name, cmap, hhea, hmtx].some(value => value === undefined)) throw new Error("字体表缺失");
const names = {};
for (let i = 0; i < data.readUInt16BE(name + 2); i++) { const p = name + 6 + i * 12; const id = data.readUInt16BE(p + 6); if (data.readUInt16BE(p) !== 3 || ![1, 4, 6].includes(id)) continue; const start = name + data.readUInt16BE(name + 4) + data.readUInt16BE(p + 10); const length = data.readUInt16BE(p + 8); names[id] = Buffer.from(data.subarray(start, start + length)).swap16().toString("utf16le"); }
for (const [id, value] of Object.entries(names)) if (meta.names[id] !== value) throw new Error("字体 family 不匹配");
function glyph(code) {
  for (let i = 0; i < data.readUInt16BE(cmap + 2); i++) {
    const sub = cmap + data.readUInt32BE(cmap + 4 + i * 8 + 4);
    if (data.readUInt16BE(sub) !== 4) continue;
    const count = data.readUInt16BE(sub + 6) / 2;
    const ends = sub + 14, starts = ends + count * 2 + 2, deltas = starts + count * 2, offsets = deltas + count * 2;
    for (let j = 0; j < count; j++) {
      if (code < data.readUInt16BE(starts + j * 2) || code > data.readUInt16BE(ends + j * 2)) continue;
      const delta = data.readInt16BE(deltas + j * 2), offset = data.readUInt16BE(offsets + j * 2);
      const value = offset ? data.readUInt16BE(offsets + j * 2 + offset + (code - data.readUInt16BE(starts + j * 2)) * 2) : code;
      return value ? (value + delta) & 65535 : 0;
    }
  }
  return 0;
}
const codes = [0x41, 0xe0b0, 0xf017, 0xf120];
const metrics = data.readUInt16BE(hhea + 34);
const widths = codes.map(code => { const id = glyph(code); if (!id) throw new Error(`字体缺少 U+${code.toString(16)}`); return data.readUInt16BE(hmtx + Math.min(id, metrics - 1) * 4); });
if (new Set(widths).size !== 1) throw new Error("Nerd Mono 图形与文本不等宽");
console.log(`字体 family=${names[1]}；SHA256、NL 发布文件、Powerline/时钟/终端 glyph 与等宽检查通过。`);
