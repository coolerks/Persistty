import { expect, it } from "vitest";
import { fileLanguages, isFileLanguage, languageForFile } from "./file-language";

it.each([
  ["Main.java", "java"], ["lib.rs", "rust"], ["a.c", "c"], ["a.cpp", "cpp"], ["a.cs", "csharp"],
  ["index.php", "php"], ["a.rb", "ruby"], ["main.tf", "hcl"], ["a.graphql", "graphql"],
  ["a.ps1", "powershell"], ["a.mdx", "mdx"], ["A.YAML", "yaml"], ["路径/a.yml", "yaml"],
  ["Dockerfile", "dockerfile"], [".editorconfig", "ini"], ["view.html.liquid", "liquid"],
  ["a.d.ts", "typescript"], [".gitconfig", "ini"], ["README.unknown", "plaintext"],
  ["a.ftl", "freemarker2"], ["schema.sql", "sql"], ["a.sv", "systemverilog"], ["a.v", "verilog"],
])("完整元数据识别 %s 为 %s", (path, expected) => { expect(languageForFile(path)).toBe(expected); });

it("无后缀 shebang 使用上游首行规则，不覆盖已知扩展", () => {
  expect(languageForFile("script", "#!/usr/bin/env python3\nprint(1)")).toBe("python");
  expect(languageForFile("script", "#!/usr/bin/env node\nconst a = 1;")).toBe("javascript");
  expect(languageForFile("script.go", "#!/usr/bin/env python3")).toBe("go");
});

it("完整 91 个模式包含无独立后缀变体并拒绝未知 ID", () => {
  expect(fileLanguages).toHaveLength(91);
  for (const id of ["freemarker2.tag-angle.interpolation-bracket", "mysql", "pgsql", "redshift", "json", "plaintext", "proto", "sol", "aes"]) expect(isFileLanguage(id)).toBe(true);
  expect(isFileLanguage("unknown")).toBe(false);
});
