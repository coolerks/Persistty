# Monaco 完整语言高亮

> 归档状态：2026-09-30 用户明确要求标记完成并归档；本任务已完成收尾。高亮/图标及相关修复已提交 6cf29aa，终端几何修复已提交 b7ac848；既有验收限制继续保留。

## 目标与背景

用户要求当前桌面编辑器覆盖其列出的全部 Monaco 语言。现有完整引擎已导入，但 `DesktopEditor.tsx:33-37` 的手写白名单将其他文件强制设为 plaintext。依据见[研究记录](../09-30-editor-language-material-icons/research/upstream-assets.md)。

## 需求

- L01：覆盖下列全部 91 个 ID，保持锁定 Monaco 0.57.0 的语法定义与按需加载。
- L02：按完整注册元数据自动识别精确文件名、最长后缀、首行解释器提示，未知格式回退 plaintext；basename/后缀按小写匹配，Unicode 原名保留。
- L03（已批准）：桌面文件内容栏提供“语言模式”选择，包含“自动识别”和全部 ID。无独立后缀的 FreeMarker 变体与 MySQL/PostgreSQL/Redshift 可手动选择；普通 `.sql` 默认 sql，`.ftl/.ftlh/.ftlx` 默认 freemarker2。
- L04（已批准）：手动选择仅改变当前浏览器文件视图高亮，按 projectId/folderId/path 区分，跨同文件左右分组与标签切换保留；关闭最后一个对应文件视图后清除，刷新页面重新自动识别。不写服务器、不修改正文或命名。
- L05：保持只读文本、当前 worker、禁诊断/LSP、桌面懒加载及手机基础文本视图；主题/模式切换不重建 model、重置光标或滚动。

## 完整语言清单

```text
plaintext
abap
apex
azcli
bat
bicep
cameligo
clojure
coffeescript
c
cpp
csharp
csp
css
cypher
dart
dockerfile
ecl
elixir
flow9
fsharp
freemarker2
freemarker2.tag-angle.interpolation-dollar
freemarker2.tag-bracket.interpolation-dollar
freemarker2.tag-angle.interpolation-bracket
freemarker2.tag-bracket.interpolation-bracket
freemarker2.tag-auto.interpolation-dollar
freemarker2.tag-auto.interpolation-bracket
go
graphql
handlebars
hcl
html
ini
java
javascript
julia
kotlin
less
lexon
lua
liquid
m3
markdown
mdx
mips
msdax
mysql
objective-c
pascal
pascaligo
perl
pgsql
php
pla
postiats
powerquery
powershell
proto
pug
python
qsharp
r
razor
redis
redshift
restructuredtext
ruby
rust
sb
scala
scheme
scss
shell
sol
aes
sparql
sql
st
swift
systemverilog
verilog
tcl
twig
typescript
typespec
vb
wgsl
xml
yaml
json
```

## 验收标准

- AC-L01：实际 Monaco getLanguages() 包含以上完整集合；每种非 plaintext 语言的 tokenizer 可加载并用合法样例产生预期类型 token，不能只检查 ID 已注册。
- AC-L02：Java/Rust/C/C++/C#/PHP/Ruby/HCL/GraphQL/PowerShell/MDX、yaml/yml、Dockerfile/.editorconfig、复合后缀、大小写、隐藏文件、Unicode、未知类型与无后缀 shebang 自动识别正确；自动识别结果来自完整 metadata，无重复维护的小白名单。
- AC-L03：键盘可打开选择并选模式，显示当前模式；FreeMarker 六个变体与三种 SQL 方言可直接使用；切回自动恢复推断，选择文件 A 不污染文件 B 或其他项目同名文件。
- AC-L04：切标签、左右分组、主题和语言后 model 身份、内容、选择及滚动保持；零文件写入请求，不启动 LSP；手机保留现有文本视图且无页面溢出。
- AC-L05：lint/typecheck/test/build 与新增针对实际行为的浏览器用例通过；跳过或环境缺失不能称为验收。

## 范围外

语义分析、补全服务/LSP、编辑保存/草稿、内容大规模自动猜测、手机 Monaco、永久文件关联设置和自定义语言。产品边界与父任务一致。

## 实施状态

PRD/设计/实施清单已备齐，语言选择与视图记忆策略已批准，用户已于 2026-09-30 明确批准开始执行；实施与验证结果见[验收报告](../09-30-editor-language-material-icons/check-report.md)。
