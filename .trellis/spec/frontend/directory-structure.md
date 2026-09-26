# React 目录结构

初始约定（尚未创建）如下；不提前创建空 feature。

```text
web/src/app/                 入口、providers、IDE layout
web/src/features/auth/       登录
web/src/features/workspaces/ 工作区切换
web/src/features/explorer/   文件树、菜单、watcher UI
web/src/features/editor/     tabs、Monaco、draft、diff
web/src/features/terminal/   tabs、xterm、attach
web/src/features/search/     search/replace preview
web/src/features/transfer/   upload/conflict/resume
web/src/features/git/        只读 Git views
web/src/features/settings/   theme/layout
web/src/components/ui/       shadcn 基础组件
web/src/lib/api/             fetch client、DTO decoder
web/src/lib/ws/              共享 WS transport
web/src/assets/              字体、图标许可清单
```

feature 内按需 components/hooks/store/api/types；只有多 feature 共用且同一概念才上移，不因两段 JSX 相似建立万能组件。app 编排 features；feature 不依赖 app；components/ui 不依赖业务；避免跨 feature 深路径 import，通过明确公共入口共享。
组件文件 PascalCase.tsx，hook useX.ts，其他文件 kebab-case.ts，测试就近 *.test.ts(x)，E2E 在根 tests/e2e。业务界面用简体中文；Close Terminal 保留明确操作名称并带中文警告。测试 fixtures 按 owner 放，不复制第二份协议类型。
