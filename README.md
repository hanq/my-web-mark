# My Web Mark

基于 **Wails v2 + Vue 3 + Three.js** 的 Windows 桌面端「书签 + 账号密码」整理工具。
以 2.5D 主题岛台的形式管理浏览器书签，密码数据本地加密存储。

![build](https://img.shields.io/badge/wails-v2.14-blue) ![go](https://img.shields.io/badge/go-1.25%2B-00ADD8) ![vue](https://img.shields.io/badge/vue-3.5-42B883)

## 功能

- **浏览器书签导入**：检测并解析 Edge / Chrome / Zen / Firefox 本地书签，按文件夹映射为主题
- **运行检测**：识别当前正在运行的浏览器及可导入书签数量
- **拖拽收集**：从浏览器拖动链接到窗口即可入库（自动抓取标题、去重）
- **2.5D 空间**：Three.js 等轴视角场景，每个主题一座岛台、书签为卡片；滚轮缩放、拖拽平移、点击聚焦
- **账号密码管理**：按主题录入站点账号，明文仅在管理员解锁后可见，复制后 15 秒自动清空剪贴板
- **管理员密码**：argon2id 校验 + AES-256-GCM 加密，改密自动全量重加密
- **三套皮肤**：暗夜科技 / 极简白 / 赛博霓虹，CSS 与 3D 场景配色联动
- **列表视图**：传统 2D 列表视图兜底，全部功能可用

## 开发

```bash
wails dev      # 开发模式（前后端热重载）
wails build    # 生产构建 -> build/bin/my-web-mark.exe
go test ./...  # 后端测试
```

## 数据与安全

- 数据库：`%APPDATA%/my-web-mark/mywebmark.db`（SQLite，WAL 模式）
- 密码派生：argon2id（64MB / 1 轮 / 4 线程），派生密钥仅存内存，锁定即清零
- 密文格式：AES-256-GCM（nonce||ciphertext），管理员密码永不明文落盘

## 目录

```
app/              Go 绑定服务（Vault/Theme/Bookmark/Account/Browser）
internal/store    SQLite DAO（modernc.org/sqlite，免 CGO）
internal/crypto   argon2id + AES-GCM
internal/bookmarks 各浏览器书签解析 + 进程检测
frontend/src      Vue3 + Pinia + Three.js 前端
spec.md           产品/技术规格
task/plan.md      任务计划与决策记录
```
