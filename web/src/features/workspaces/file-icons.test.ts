import { expect, it } from "vitest";
import { defaultFileIcon, fileIconFor, fileIconURL } from "./file-icons";

it.each([
  ["a.yaml", "yaml"], ["a.yml", "yaml"], ["a.yaml.dist", "yaml"], ["a.yml.dist", "yaml"],
  ["a.d.ts", "typescript-def"], ["a.ts", "typescript"], ["a.tsx", "react_ts"],
  ["package.json", "nodejs"], ["go.mod", "go-mod"], ["Dockerfile", "docker"],
  [".gitignore", "git"], [".env", "tune"], ["docker-compose.yml", "docker"],
  ["docker-compose.yaml", "docker"], ["目录/CONFIG.YML", "yaml"],
])("Material 上游映射 %s → %s", (path, expected) => { expect(fileIconFor({ path, theme: "dark" })).toBe(expected); });

it("未知类型与原型属性名只使用 plaintext 内置语言回退，不构造用户资源路径", () => {
  for (const path of ["未知.blob", "constructor", "__proto__", "toString"]) {
    expect(fileIconFor({ path })).toBe("document");
    expect(fileIconURL(fileIconFor({ path }))).toMatch(/^\/material-icons\/[\w-]+\.svg$/);
  }
});

it("上游目录开闭、根目录与浅色覆盖独立解析", () => {
  expect(fileIconFor({ path: "src", kind: "directory", theme: "dark" })).toBe("folder-src");
  expect(fileIconFor({ path: "src", kind: "directory", expanded: true, theme: "dark" })).toBe("folder-src-open");
  expect(fileIconFor({ path: ".git", kind: "directory", theme: "dark" })).toBe("folder-git");
  expect(fileIconFor({ path: "node_modules", kind: "directory", theme: "dark" })).toBe("folder-node");
  expect(fileIconFor({ path: "unknown", kind: "root", expanded: true })).toBe(defaultFileIcon({ path: "unknown", kind: "root", expanded: true }));
  expect(fileIconFor({ path: "config.toml", theme: "light" })).toBe("toml_light");
  expect(fileIconFor({ path: "config.toml", theme: "dark" })).toBe("toml");
});
