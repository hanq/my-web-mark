# My Web Mark

本地优先的 Windows 桌面端「书签 + 密码」整理工具：用 2.5D 卡片画廊浏览书签，按主题归类，密码在本地加密存储，不上云、不联网同步。

![wails](https://img.shields.io/badge/wails-v2.14-blue) ![go](https://img.shields.io/badge/go-1.26%2B-00ADD8) ![vue](https://img.shields.io/badge/vue-3.5-42B883) ![three](https://img.shields.io/badge/three.js-0.187-black)

## 为什么做这个

- 书签散落在 Edge / Chrome / Zen / Firefox 各自的文件夹里，换浏览器就丢上下文
- 账号密码零散记在便签、文档和脑子里，交给云服务又不放心
- 需要一个「本地文件即全部数据」、能按主题整理、还能一眼看清楚的工具

于是有了 My Web Mark：数据只存在本机 SQLite，密码用管理员密码派生密钥加密，界面上用 2.5D 画廊给书签一个可浏览的空间，同时保留传统列表视图兜底。

## 功能

### 书签

- **多浏览器导入**：解析 Edge / Chrome / Zen / Firefox 的本地书签，文件夹层级自动映射为主题分组，并显示浏览器是否正在运行
- **主题级导入导出**：导出 Netscape HTML（任意浏览器可直接导入）或 JSON（含标签、备注、置顶等完整信息）；导入时自动识别格式、同名主题自动加后缀
- **拖拽收集**：从浏览器把链接拖进窗口即可入库（2D / 2.5D 视图都支持），自动抓取页面标题并按 URL 去重
- **2.5D 卡片画廊**：正交相机 + Cover Flow 布局，滚轮 / 左右方向键 / 拖拽切换，点击卡片 360° 翻转后打开；渐变天空球、舞台光斑与 Bloom 辉光，亮度刻意压弱以免干扰阅读
- **列表视图**：自适应多列网格，`0`–`9` 数字键快速打开对应书签，卡片拖拽排序（置顶组内），favicon 探测缓存、失败回退首字母
- **标签体系**：主题内可建多个标签，书签支持多标签；列表与 2.5D 视图都有筛选条，另有标签管理对话框
- **主题侧栏**：23 个 IconPark 风格线性图标可选、拖拽排序、右键菜单（编辑 / 导出 / 删除）、可收起（收起只留图标）

### 账号与密码

- **不强制解锁**：浏览条目不需要密码，只有查看 / 复制明文时才弹解锁框
- **条目类型**：网站账号、设备密码（开机、电脑解锁）、WiFi 密码、其他；网站类条目可以不填站点名
- **关联书签**：可把账号关联到当前主题的书签，自动带出站点名与网址；没关联时按 URL 域名推导站点名
- **密码可选**：只记账号、只记 WiFi 名称、只写备注都能保存，不强制填密码
- **标签与搜索**：账号同样支持主题内标签和关键字搜索，和书签的筛选互不干扰
- **三种复制方式**：复制用户名、复制密码、复制整条概要（名称 / 用户名 / 密码 / 网址 / 备注 / 标签，空字段自动省略）；敏感内容复制后 15 秒自动清空剪贴板
- **卡片布局**：站点信息、用户名、密码分栏展示，密码默认掩码；复制密码不会把明文展开到界面

### 安全

- 管理员密码用 **argon2id**（64 MB / 1 轮 / 4 线程）派生密钥，只存哈希与 salt
- 账号密码用 **AES-256-GCM**（12 字节随机 nonce + 密文）存储，数据库文件单独泄露也无法还原明文
- 派生密钥只存在内存，锁定即清零；修改管理员密码会全量重新加密
- 未解锁状态下，API 层拒绝返回任何明文敏感字段

### 界面与系统

- **8 套皮肤**：暗夜科技 / 极简白 / 赛博霓虹 / 黑板粉笔 / 黑曜石 / 竹子 / 玉石 / 彩虹，CSS 主题变量与 3D 场景配色同步切换
- **无边框自绘标题栏**：拖拽移动、双击最大化、窗口控制按钮；记住窗口位置与大小，副屏断开时自动回退主屏并夹紧越界坐标
- **记住最后视图**：启动时回到上次使用的列表或 2.5D 空间
- **单实例运行**：重复启动会唤起已打开的窗口
- **数据库自动备份**：启动时用 `VACUUM INTO` 做一致性快照，保留最近 5 份
- **书签打开方式**：跟随系统默认，或指定 Edge / Chrome / Zen / Firefox（自动探测安装路径，未安装则回退系统默认）

## 技术栈

| 层 | 选型 |
|----|------|
| 桌面壳 | Wails v2.14（Windows WebView2） |
| 后端 | Go 1.26、`modernc.org/sqlite`（纯 Go，免 CGO）、`golang.org/x/crypto`（argon2） |
| 前端 | Vite 7 + Vue 3.5 + TypeScript + Pinia + Vue Router |
| 3D | Three.js 0.187（正交相机、CanvasTexture、后处理 Bloom） |
| 存储 | SQLite 单文件 + WAL，数据库与备份都在用户目录下 |

## 快速开始

环境要求：Go 1.26+、Node.js 20+、Wails CLI v2.14+（`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）、Windows 10/11（WebView2 运行时）。

```bash
wails dev      # 开发模式，前端热重载
wails build    # 生产构建，产物 build/bin/my-web-mark.exe
```

单独构建前端或跑测试：

```bash
cd frontend && npm install && npm run build   # 含 vue-tsc 类型检查
go test ./...                                 # 后端测试
```

## 数据与安全

| 内容 | 位置 / 说明 |
|------|-------------|
| 数据库 | `%APPDATA%/my-web-mark/mywebmark.db`（SQLite，WAL 模式） |
| 备份 | `%APPDATA%/my-web-mark/backup/`，启动时快照，保留最近 5 份 |
| favicon 缓存 | `%APPDATA%/my-web-mark/favicons/`（按 host 哈希命名，失败负缓存 10 分钟） |
| 窗口状态 | `%APPDATA%/my-web-mark/` 下的状态文件，多屏变化时自动降级 |
| 密码派生 | argon2id（64 MB / 1 轮 / 4 线程），密钥仅内存持有 |
| 密文格式 | AES-256-GCM，`nonce ∥ ciphertext` |

删除应用只需退出并移除 `%APPDATA%/my-web-mark/` 目录；卸载前请确认已备份需要保留的数据。

## 项目结构

```
app/                 Wails 绑定层（Vault / Theme / Tag / Bookmark / Account / Browser / Settings / Export / Favicon）
internal/store       SQLite DAO、迁移、备份、设置 KV
internal/crypto      argon2id 口令派生 + AES-GCM
internal/bookmarks   各浏览器书签解析（Chromium JSON / places.sqlite / Netscape HTML）与进程检测
internal/favicon     favicon 抓取、磁盘缓存、HTTP 分发
internal/appstate    窗口状态持久化与多屏降级校验
frontend/src         Vue 3 前端（views 页面、scenes 2.5D 引擎、stores 状态、components 组件）
frontend/wailsjs     Wails 自动生成的绑定与运行时类型
build/               图标、安装器脚本与构建产物（build/bin 不入库）
spec.md              产品与技术规格
task/plan.md         任务计划、决策记录与增量迭代清单
```

## 测试

```bash
go test ./...        # 30 个后端测试：加解密、存储、书签解析、favicon、账号/金库流程
cd frontend && npx vue-tsc --noEmit   # 前端类型检查
```

## 边界

- 不做云同步、不做浏览器扩展、不抓取网页正文
- 不做浏览器表单自动填充
- 目前只支持 Windows；Firefox / Zen 只读取书签，不分析历史记录
- 账号密码完全依赖管理员密码保护，忘记管理员密码无法找回（可删除 `vault_meta` 重置，代价是已存密码失效）
