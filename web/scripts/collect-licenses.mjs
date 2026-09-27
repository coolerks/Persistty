import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, readdirSync, writeFileSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath, URL } from "node:url";

const root = fileURLToPath(new URL("../", import.meta.url));
const tree = JSON.parse(execFileSync("npm", ["ls", "--omit=dev", "--all", "--long", "--json"], {
  cwd: root, encoding: "utf8", maxBuffer: 32 * 1024 * 1024,
}));
const seen = new Set();
const notices = [];

function visit(node) {
  if (node.path && node.path !== root.replace(/\/$/, "") && !seen.has(node.path)) {
    seen.add(node.path);
    const files = readdirSync(node.path).filter(name => /^(license|licence|copying|notice)(\.|$)/i.test(name)).sort();
    const licenses = files.filter(name => /^(license|licence|copying)(\.|$)/i.test(name));
    const contents = files.map(name => `${name}\n${readFileSync(join(node.path, name), "utf8")}`);
    if (!licenses.length) {
      // This tarball omits LICENSE; use the documented upstream grant only for this version.
      if (node.name === "react-remove-scroll-bar" && node.version === "2.3.8")
        contents.unshift(readFileSync(join(root, "licenses/react-remove-scroll-bar-2.3.8.txt"), "utf8"));
      else throw new Error(`运行依赖缺少许可证，停止构建：${node.name}@${node.version}`);
    }
    if (!contents.every(content => content.trim().length > 0)) throw new Error(`空许可证：${node.name}`);
    notices.push({ name: `${node.name}@${node.version}`, text: contents.join("\n\n") });
  }
  for (const child of Object.values(node.dependencies ?? {})) visit(child);
}
visit(tree);
notices.push({ name: "shadcn/ui 复制组件", text: readFileSync(join(root, "licenses/shadcn-ui.txt"), "utf8") });
notices.sort((a, b) => a.name.localeCompare(b.name, "en"));
mkdirSync(join(root, "public"), { recursive: true });
writeFileSync(join(root, "public/third-party-licenses.txt"), notices.map(notice => `${notice.name}\n${"=".repeat(72)}\n${notice.text}`).join("\n\n"));
console.log(`已保留 ${notices.length} 项第三方许可证与通知。`);
