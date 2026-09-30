import { randomUUID } from "node:crypto";
import { existsSync, readFileSync, renameSync, writeFileSync } from "node:fs";
import { Buffer } from "node:buffer";

// Unchanged output does not trigger HMR; changed output is never visible half-written.
export function writeGeneratedFile(path, data) {
  const bytes = Buffer.isBuffer(data) ? data : Buffer.from(data);
  if (existsSync(path) && readFileSync(path).equals(bytes)) return;
  const temporary = `${path}.${randomUUID()}.tmp`;
  writeFileSync(temporary, bytes);
  renameSync(temporary, path);
}
