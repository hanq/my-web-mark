# 任务计划与进度

> 状态标记：[ ] 未开始 · [~] 进行中 · [x] 完成

## 0. 技术可行性结论

| 风险点 | 结论 | 对策 |
|--------|------|------|
| Wails v2 + Three.js 2.5D | ✅ 可行 | WebView2 支持 WebGL；用正交相机+等轴布局降低 3D 复杂度 |
| 纯 Go SQLite（免 CGO） | ✅ 可行 | modernc.org/sqlite 纯 Go 实现，Wails 构建无 CGO 负担 |
| Firefox places.sqlite 被锁 | ✅ 可行 | 复制到临时目录再打开（SQLite 只读） |
| Zen 浏览器识别 | ✅ 可行 | Zen 基于 Firefox，Profile 目录含 `zen` 标识；进程名 zen.exe；书签同 places.sqlite |
| 拖拽 URL 入窗 | ✅ 可行 | WebView2 支持 HTML5 drop 事件，`e.dataTransfer.getData('text/uri-list')` |
| 进程检测 | ✅ 可行 | `gopsutil/v3/process` 列进程名匹配浏览器 |
| 密码加密 | ✅ 可行 | golang.org/x/crypto argon2 + AES-GCM 标准库 |
| 切主题联动 3D | ✅ 可行 | Three.js 场景参数对象化，CSS variables 同步 |

## 1. 任务列表

### 阶段一：项目骨架
- [ ] T1 编写 AGENTS.md / spec.md / task/plan.md
- [x] T2 wails init 生成项目骨架（Vue+TS 模板）
- [x] T3 引入 modernc.org/sqlite，DB 初始化与迁移（theme/bookmark/account/vault_meta）
- [x] T4 internal/crypto：argon2id 哈希 + AES-256-GCM 加解密 + 单元测试

### 阶段二：Go 业务层
- [x] T5 internal/bookmarks：Chromium JSON 解析器（Edge/Chrome）+ 单测
- [x] T6 internal/bookmarks：Firefox/Zen places.sqlite 解析器 + 进程检测
- [x] T7 app 层绑定：ThemeService / BookmarkService / AccountService / VaultService
- [x] T8 拖拽导入 + URL 元数据抓取（title 提取）辅助 API

### 阶段三：前端基础
- [x] T9 Pinia stores（theme/bookmark/account/vault）+ 路由骨架
- [x] T10 主题化样式系统（CSS variables，3 套主题，SVG 图标库）
- [x] T11 2D 列表视图：主题管理 + 书签 CRUD + 账号 CRUD（含解锁流程）
- [x] T12 浏览器书签导入向导 UI + 打开书签检测面板

### 阶段四：2.5D 场景
- [x] T13 Three.js 场景引擎封装（正交相机、灯光、地面网格、主题岛屿布局）
- [x] T14 书签卡片 3D 渲染（CanvasTexture 文字、悬停/点击/拖拽）
- [x] T15 相机聚焦动画 + 场景与主题配色联动

### 阶段五：收尾
- [x] T16 wails build 生产构建验证
- [x] T17 README 使用说明

## 2. 决策记录

- 2026-XX: 选 Vue3 而非 React——wails 官方模板成熟、团队熟悉
- 2026-XX: 选 modernc.org/sqlite 而非 mattn/go-sqlite3——避免 CGO，交叉编译友好
- 2026-XX: Zen 按 Firefox 分支处理（profile 目录名含 "zen"）
- 2026-XX: 2.5D 用正交相机 + 斜 45° 布局实现，比真透视更稳、信息密度更高
