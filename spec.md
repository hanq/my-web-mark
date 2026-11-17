# my-web-mark 规格说明（spec）

## 1. 产品定位

一款 Windows 桌面端「书签 + 账号密码」整理工具。以 2.5D 空间化的方式管理浏览器书签与网站账号，按「主题」（如：开发、设计、影音、工作）组织，敏感信息由管理员密码保护，数据全部本地化。

## 2. 核心功能

### 2.1 书签管理

| 编号 | 功能 | 说明 |
|------|------|------|
| F1 | 浏览器书签导入 | 支持解析 Edge / Chrome / Zen / Firefox 的本地书签文件，按文件夹映射为主题分组 |
| F2 | 打开书签检测 | 监视当前正在运行的浏览器进程及其最近打开的书签页（读取 Bookmarks/History 中的最近条目与进程列表） |
| F3 | 拖拽收集 | 从浏览器拖动 URL 到应用窗口即可入库（HTML5 drag & drop + 系统剪贴板兜底） |
| F4 | 主题书签列表 | 书签按主题组织；2.5D 场景中每个主题是一个"岛屿/展台"，书签为台面上的卡片 |
| F5 | 书签编辑 | 新增/改名/换主题/删除/置顶/备注 |

### 2.2 账号密码管理

| 编号 | 功能 | 说明 |
|------|------|------|
| F6 | 按主题管理账号 | 每个主题下可录入多条账号（站点、用户名、密码、备注、URL） |
| F7 | 管理员密码 | 首次启动设置；解锁后可查看/复制明文密码；锁定状态仅显示掩码 |
| F8 | 密码加密存储 | argon2id 派生密钥 + AES-256-GCM 加密；DB 泄露不可还原明文 |
| F9 | 一键复制 | 复制用户名/密码到剪贴板（15s 后自动清空剪贴板可选） |

### 2.3 2.5D 界面

| 编号 | 功能 | 说明 |
|------|------|------|
| F10 | 主题岛屿 | Three.js 正交相机 + 等轴透视风格的 2.5D 场景；每个主题一个可点击的平台 |
| F11 | 书签卡片 | 书签渲染为 3D 面片卡片，支持悬停高亮、点击打开、拖拽换位 |
| F12 | 相机交互 | 滚轮缩放、右键/空格+拖拽平移、点击主题聚焦（相机平滑过渡） |
| F13 | 列表视图切换 | 3D 场景与传统 2D 列表视图一键切换，保证可用性兜底 |

### 2.4 主题化风格

| 编号 | 功能 | 说明 |
|------|------|------|
| F14 | 换肤系统 | CSS variables + `data-theme`；内置 暗夜科技 / 极简白 / 赛博霓虹 三主题 |
| F15 | 3D 配色联动 | 切换主题时同步更新 Three.js 场景的背景、雾色、材质颜色 |

## 3. 技术架构

```
┌─────────────── Wails 窗口 ───────────────┐
│  Frontend (Vite+Vue3+TS+Three.js)        │
│  ├─ 2.5D 场景 (Three.js Ortho + Raycast) │
│  ├─ 2D 列表视图 (Vue 组件)               │
│  ├─ Pinia stores (bookmarks/accounts/    │
│  │   vault/theme)                        │
│  └─ wailsjs bindings (自动生成)          │
└──────────────┬───────────────────────────┘
               │ JSON-RPC (window.go.*)
┌──────────────┴───────────────────────────┐
│  Go app.App 绑定层                        │
│  ├─ BookmarkService  导入/检测/CRUD       │
│  ├─ AccountService   账号 CRUD + 解锁态   │
│  └─ VaultService     管理员密码/派生密钥   │
├──────────────────────────────────────────┤
│  internal/store   modernc.org/sqlite DAO │
│  internal/bookmarks 各浏览器书签解析      │
│  internal/crypto  argon2id + AES-GCM     │
└──────────────────────────────────────────┘
```

### 3.1 数据库设计

```sql
-- 主题
CREATE TABLE theme (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#4f8cff',
    icon TEXT,             -- SVG 名或图标标识
    sort INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 书签
CREATE TABLE bookmark (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    theme_id INTEGER NOT NULL REFERENCES theme(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    url TEXT NOT NULL,
    favicon TEXT,          -- favicon url 或 data-uri
    note TEXT DEFAULT '',
    pinned INTEGER NOT NULL DEFAULT 0,
    source TEXT DEFAULT 'manual',  -- manual/edge/chrome/zen/firefox
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(theme_id, url)
);

-- 账号（密码为密文）
CREATE TABLE account (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    theme_id INTEGER NOT NULL REFERENCES theme(id) ON DELETE CASCADE,
    site TEXT NOT NULL,
    username TEXT NOT NULL,
    password_cipher BLOB NOT NULL,   -- AES-256-GCM(nonce||ct)
    url TEXT DEFAULT '',
    note TEXT DEFAULT '',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 管理员凭据（单行）
CREATE TABLE vault_meta (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    salt BLOB NOT NULL,
    password_hash BLOB NOT NULL,     -- argon2id
    kdf_params TEXT NOT NULL,        -- JSON: time,memory,threads
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 3.2 浏览器书签路径（Windows）

| 浏览器 | Bookmarks 文件 |
|--------|----------------|
| Edge | `%LOCALAPPDATA%/Microsoft/Edge/User Data/Default/Bookmarks` |
| Chrome | `%LOCALAPPDATA%/Google/Chrome/User Data/Default/Bookmarks` |
| Zen | `%APPDATA%/zen-browser-data/Profiles/*/bookmark.json` 或 Firefox 风格 `places.sqlite` |
| Firefox | `%APPDATA%/Mozilla/Firefox/Profiles/*/places.sqlite`（只读复制后查询） |

- Chromium 系为 JSON，直接解析 `roots.bookmark_bar/other/synced` 树
- Firefox/Zen 的 places.sqlite 被进程占用，采用「复制临时文件后打开」策略
- 检测运行中浏览器：`tasklist` 或 Go `gopsutil` 查进程名（msedge.exe/chrome.exe/zen.exe/firefox.exe）

### 3.3 加密方案

1. 首次启动：生成 16B salt → argon2id(adminPwd, salt, time=1, mem=64MB, threads=4) → 存 hash+salt
2. 解锁：argon2id 校验 hash → 用派生 key 前 32B 作为 AES-256-GCM 主密钥（内存持有，锁定即清零）
3. 账号密码存储：12B 随机 nonce + ciphertext + tag，整块存 `password_cipher`
4. 锁定：置零密钥字节，API 层返回掩码

## 4. 非功能需求

- 数据文件全本地，无网络请求（favicon 用本地缓存，加载失败显示首字母占位）
- 单实例运行（wails 单实例锁）
- 书签 URL 去重（theme+url 唯一约束）
- 列表视图保证无 WebGL 环境也可用
- DB 定期备份到 `backup/mywebmark-YYYYMMDD.db`（保留最近 5 份）

## 5. 边界与不做的事

- 不做云同步、不做浏览器扩展、不抓取网页内容
- 不做自动填充浏览器表单
- Firefox 历史记录只读取书签表（moz_places + moz_bookmarks），不分析全部历史
