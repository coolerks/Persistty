import { mkdirSync, readFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath, URL } from "node:url";
import ts from "typescript";
import { writeGeneratedFile } from "./write-generated-file.mjs";

const root = fileURLToPath(new URL("../", import.meta.url));
const esm = resolve(root, "node_modules/monaco-editor/esm/vs");
const version = JSON.parse(readFileSync(resolve(root, "node_modules/monaco-editor/package.json"), "utf8")).version;
if (version !== "0.57.0") throw new Error("Monaco 升级后须重新核对语言元数据生成契约");

function source(path) {
  return ts.createSourceFile(path, readFileSync(path, "utf8"), ts.ScriptTarget.Latest, true, ts.ScriptKind.JS);
}

function registrations(path, callee) {
  const ast = source(path);
  const constants = new Map();
  const result = [];
  function literal(node) {
    if (ts.isStringLiteral(node)) return node.text;
    if (ts.isArrayLiteralExpression(node)) return node.elements.map(literal);
    if (ts.isIdentifier(node) && constants.has(node.text)) return literal(constants.get(node.text));
    if (ts.isCallExpression(node) && node.expression.getText(ast) === "localize" && ts.isStringLiteral(node.arguments[1])) return node.arguments[1].text;
    throw new Error(`非静态语言元数据：${path}: ${node.getText(ast)}`);
  }
  function visit(node) {
    if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.initializer) constants.set(node.name.text, node.initializer);
    if (ts.isCallExpression(node) && node.expression.getText(ast) === callee) {
      const object = node.arguments[0];
      if (!object || !ts.isObjectLiteralExpression(object)) throw new Error(`语言注册不是静态对象：${path}`);
      const entry = {};
      for (const property of object.properties) {
        if (!ts.isPropertyAssignment(property)) throw new Error(`语言注册字段格式变化：${path}`);
        const key = property.name.getText(ast);
        if (["id", "aliases", "extensions", "filenames", "firstLine"].includes(key)) entry[key] = literal(property.initializer);
      }
      if (typeof entry.id !== "string") throw new Error(`缺语言 ID：${path}`);
      result.push(entry);
    }
    ts.forEachChild(node, visit);
  }
  visit(ast);
  if (!result.length) throw new Error(`没有语言注册：${path}`);
  return result;
}

const entries = registrations(resolve(esm, "editor/common/languages/modesRegistry.js"), "ModesRegistry.registerLanguage");
const main = source(resolve(esm, "editor/editor.main.js"));
for (const statement of main.statements) {
  if (!ts.isImportDeclaration(statement) || !ts.isStringLiteral(statement.moduleSpecifier)) continue;
  const path = statement.moduleSpecifier.text;
  if (path.startsWith("../languages/definitions/") && path.endsWith("/register.js")) entries.push(...registrations(resolve(esm, "editor", path), "registerLanguage"));
}
entries.push(...registrations(resolve(esm, "languages/features/json/register.js"), "languages.register"));
if (entries.length !== 91 || new Set(entries.map(entry => entry.id)).size !== 91) throw new Error("语言集合变化，须重新核对完整支持清单");
const destination = resolve(root, "src/generated/languages.json");
mkdirSync(dirname(destination), { recursive: true });
writeGeneratedFile(destination, `${JSON.stringify({ version, languages: entries }, null, 2)}\n`);
console.log(`已生成 Monaco ${version} 的 ${entries.length} 个语言模式。`);
