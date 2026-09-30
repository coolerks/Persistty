import { createHash } from "node:crypto";
import { mkdirSync, readFileSync, readdirSync, realpathSync, rmSync } from "node:fs";
import { dirname, resolve, sep } from "node:path";
import { fileURLToPath, URL } from "node:url";
import { generateManifest } from "material-icon-theme";
import { writeGeneratedFile } from "./write-generated-file.mjs";

const root = fileURLToPath(new URL("../", import.meta.url));
const packageRoot = resolve(root, "node_modules/material-icon-theme");
const packageInfo = JSON.parse(readFileSync(resolve(packageRoot, "package.json"), "utf8"));
const lock = JSON.parse(readFileSync(resolve(root, "package-lock.json"), "utf8")).packages["node_modules/material-icon-theme"];
const source = JSON.parse(readFileSync(resolve(root, "licenses/material-icon-theme-source.json"), "utf8"));
if (packageInfo.version !== source.version || lock.integrity !== source.integrity) throw new Error("Material Icon Theme 版本或校验值变化，须重新审查来源与许可");
const manifest = generateManifest();
const destination = resolve(root, "public/material-icons");
mkdirSync(destination, { recursive: true });
const iconRoot = realpathSync(resolve(packageRoot, "icons"));
const hashes = {};
for (const [id, definition] of Object.entries(manifest.iconDefinitions)) {
  if (!/^[a-zA-Z0-9_-]+$/.test(id)) throw new Error(`不安全的图标 ID：${id}`);
  const path = realpathSync(resolve(packageRoot, "dist", definition.iconPath));
  if (!path.startsWith(iconRoot + sep) || !path.endsWith(".svg")) throw new Error(`图标源路径超出发布目录：${id}`);
  const svg = readFileSync(path);
  writeGeneratedFile(resolve(destination, `${id}.svg`), svg);
  hashes[`${id}.svg`] = createHash("sha256").update(svg).digest("hex");
  definition.iconPath = `material-icons/${id}.svg`;
}
// Validate every association and fallback, including light/highContrast and generated clones.
const associations = ["fileNames", "fileExtensions", "languageIds", "folderNames", "folderNamesExpanded", "rootFolderNames", "rootFolderNamesExpanded"];
const defaults = ["file", "folder", "folderExpanded", "rootFolder", "rootFolderExpanded"];
function validate(theme) {
  const ids = [...associations.flatMap(key => Object.values(theme[key] ?? {})), ...defaults.flatMap(key => theme[key] ? [theme[key]] : [])];
  for (const id of ids) if (!manifest.iconDefinitions[id]) throw new Error(`关联缺少图标：${id}`);
  for (const key of ["light", "highContrast"]) if (theme[key]) validate(theme[key]);
}
validate(manifest);
const license = readFileSync(resolve(packageRoot, "LICENSE"));
const retainedLicense = readFileSync(resolve(root, "licenses/material-icon-theme.txt"));
if (!license.equals(retainedLicense)) throw new Error("Material Icon Theme 许可证变化，须重新审查");
writeGeneratedFile(resolve(destination, "LICENSE.txt"), license);
writeGeneratedFile(resolve(destination, "assets.json"), `${JSON.stringify({ ...source, modifications: "SVG 字节原样复制，manifest 路径重写为本地 URL", sha256: hashes }, null, 2)}\n`);
const metadata = resolve(root, "src/generated/material-icons.json");
mkdirSync(dirname(metadata), { recursive: true });
writeGeneratedFile(metadata, `${JSON.stringify(manifest)}\n`);
// Only remove obsolete generated SVGs after publishing all current resources.
for (const name of readdirSync(destination)) if (/^[a-zA-Z0-9_-]+\.svg$/.test(name) && !Object.hasOwn(hashes, name)) rmSync(resolve(destination, name));
console.log(`已生成 Material Icon Theme ${source.version} 的 ${Object.keys(hashes).length} 个内置 SVG 及完整映射。`);
