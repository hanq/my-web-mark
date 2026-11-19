# My Web Mark

基于 **Wails v2 + Vue 3 + Three.js** 的 Windows 桌面端「书签 + 账号密码」整理工具。
以 3D 卡片画廊（Cover Flow）的形式管理浏览器书签，密码数据本地加密存储。

![build](https://img.shields.io/badge/wails-v2.14-blue) ![go](https://img.shields.io/badge/go-1.25%2B-00ADD8) ![vue](https://img.shields.io/badge/vue-3.5-42B883)

## 功能

### 书签
- **浏览器书签导入**：检测并解析 Edge / Chrome / Zen / Firefox 本地书签，按文件夹映射为主题；运行状态检测
- **文件导入/导出**：主题级导出 HTML（Netscape 标准，浏览器可导入）或 JSON（完整备份含标签/备注/置顶）；导入自动检测格式、同名自动后缀
- **拖拽收集**：从浏览器拖动链接到窗口（2D/3D 视图均可）即可入库，自动抓取标题、去重
- **3D 卡片画廊（Cover Flow）**：中央卡正对相机、两侧透视后退；滚轮/←→/拖拽切换，点击翻转动画后打开；渐变天空球 + 地板舞台光斑 + Bloom 柔光（可调弱不干扰阅读）
- **列表视图**：自适应多列卡片网格、kbd 键帽式编号（按数字键 0–9 快捷打开）、卡片拖拽排序（置顶组内）、favicon 显示（探测缓存，失败回退首字母）
- **标签体系**：主题内多标签，书签可打多标签；列表上方 / 2.5D 下方的筛选条；标签管理对话框
- **主题侧栏**：IconPark 风格图标选择、拖拽排序、更多操作菜单（编辑/导出/删除/右键呼出）、可收起（收起仅图标 + 悬停提示）

### 账号与安全
- **账号密码管理**：按主题录入站点账号，明文仅在管理员解锁后可见，复制后 15 秒自动清空剪贴板
- **管理员密码（金库）**：argon2id 校验 + AES-256-GCM 加密；派生密钥仅存内存、锁定即清零；改密自动全量重加密；顶栏一键锁定/解锁

### 界面与系统
- **8 套皮肤**：暗夜科技 / 极简白 / 赛博霓虹 / 黑板粉笔 / 黑曜石 / 竹子 / 玉石 / 彩虹——CSS 与 3D 场景配色联动
- **无边框自绘标题栏**：拖拽移动、双击最大化、窗口控制按钮；**记住窗口位置/大小**（多屏降级校验：副屏断开回退主屏居中、越界夹紧）
- **单实例运行**：二次启动唤起已有窗口
- **DB 备份**：启动时自动备份（VACUUM INTO 一致性快照），保留最近 5 份
- **书签打开方式**：跟随系统 / 指定 Edge / Chrome / Zen / Firefox（候选路径探测，未安装自动回退系统默认）

## 开发

```bash
wails dev      # 开发模式（前后端热重载）
wails build    # 生产构建 -> build/bin/my-web-mark.exe
go test ./...  # 后端测试（29 个）
```

## 数据与安全

- 数据库：`%APPDATA%/my-web-mark/mywebmark.db`（SQLite，WAL 模式；备份在 `backup/` 子目录）
- 密码派生：argon2id（64MB / 1 轮 / 4 线程），派生密钥仅存内存，锁定即清零
- 密文格式：AES-256-GCM（nonce||ciphertext），管理员密码永不明文落盘
- favicon 缓存：`%APPDATA%/my-web-mark/favicons/`（host 哈希命名，负缓存 10 分钟）

## 目录

```
app/                Go 绑定服务（Vault/Theme/Tag/Bookmark/Account/Browser/Settings/Export/Favicon 等）
internal/store      SQLite DAO（modernc.org/sqlite，免 CGO，WAL + 备份）
internal/crypto     argon2id + AES-GCM
internal/bookmarks  浏览器书签解析（含 Netscape HTML 导入导出）+ 进程检测
internal/favicon    favicon 抓取/缓存/HTTP 分发
internal/appstate   窗口状态持久化与多屏降级校验
frontend/src        Vue3 + Pinia + Three.js 前端（scenes/ 为画廊引擎）
spec.md             产品/技术规格
task/plan.md        任务计划与增量记录
```
