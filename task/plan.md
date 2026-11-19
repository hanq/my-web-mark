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

## 3. 增量需求记录（初始 17 任务后）

全部已完成的增量功能（按会话顺序）：

- 对话框文本选择误关修复（useMaskDismiss composable）
- 无边框自绘标题栏（Frameless + --wails-draggable + 窗口控制按钮 + 记住视图模式）
- 2.5D 由环形展厅改版为 Cover Flow 卡片画廊（拖拽环视改为滚轮/方向键/拖拽切换）
- favicon 抓取缓存全栈（internal/favicon + Wails AssetServer 兜底分发）+ 列表白底 + 3D 徽标
- 主题内标签体系（tag/bookmark_tag 表 + 筛选条 2D/3D 双端 + 管理对话框）
- 书签卡片网格布局重做 + 卡片拖拽排序（bookmark.sort 列，探测式迁移）
- 金库解锁入口（顶栏按钮弹解锁框）；主题编辑入口；UpdateTheme 不再重置 sort（bug 修复）
- 设置页：书签打开方式（settings KV 表 + 浏览器候选路径探测 + 回退系统默认）
- 书签编号快捷键（0-9，kbd 键帽风格弱化，嵌入卡片内部）
- 主题书签导出/导入（Netscape HTML + JSON 双格式，自动检测，同名后缀；applyImport 可测）
- 主题图标（theme_icons 注册表 24 个 IconPark 风格图标 + ThemeDialog 选择网格）
- 侧栏收起/展开（收起仅图标 + CSS tooltip）、更多操作菜单（Teleport + 右键呼出）、底部按钮统一
- 主题侧栏拖拽排序（themeApi.sort 闭环）
- 8 套皮肤（+黑板粉笔/黑曜石/竹子/玉石/彩虹），CSS 与 3D 场景配色联动
- 3D 视觉增强：渐变天空球、地板舞台光斑、Bloom 辉光、流光、玻璃拟态卡片（favicon 高斯模糊正片叠底）、点击 360° 翻转、按压 tilt；辉光/流光两轮减弱
- 渲染质量：composer 4x MSAA render target + 纹理各向异性拉满
- 窗口状态记忆（appstate：Win32 虚拟桌面探测 + 多屏降级校验）+ 单实例锁
- DB 启动备份（VACUUM INTO，保留 5 份）
- appicon v2（几何居中 + 舞台光晕/圆环 + 面渐变 + 丝带立体）

测试规模：29 个（crypto 5 / store 1 / bookmarks 9 / favicon 4 / app 10）
