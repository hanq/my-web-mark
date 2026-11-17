# my-web-mark 项目指令

> 基于 Wails v2 的桌面端「书签 + 账号」整理工具，2.5D 主题化书签管理界面，本地 SQLite 存储。

## 项目结构约定

```
my-web-mark/
├── AGENTS.md            # 本文件：项目开发规范
├── spec.md              # 产品/技术规格说明
├── task/
│   └── plan.md          # 任务计划与进度跟踪
├── main.go              # Wails 应用入口
├── app/                 # 绑定到前端的 Go 业务层（App struct 及方法）
├── internal/
│   ├── store/           # SQLite 存储层（数据库初始化、DAO）
│   ├── bookmarks/       # 浏览器书签导入/解析/监视
│   └── crypto/          # 敏感信息加密（管理员密码派生密钥）
├── frontend/            # Wails 前端（Vite + Vue3 + Three.js）
│   └── src/
│       ├── scenes/      # 2.5D 场景相关模块
│       ├── views/       # 页面视图
│       ├── stores/      # Pinia 状态
│       └── styles/      # 主题样式
└── wails.json           # Wails 配置
```

## 技术栈（已定，勿随意更换）

- **后端**: Go + Wails v2 + `modernc.org/sqlite`（纯 Go SQLite，免 CGO）+ `golang.org/x/crypto`（argon2/pbkdf2）
- **前端**: Vite + Vue 3 + TypeScript + Pinia + Three.js（2.5D 场景）+ Vue Router
- **存储**: SQLite 单文件库，默认路径 `%APPDATA%/my-web-mark/mywebmark.db`

## 开发规范

1. **Go 代码**: 包名小写；错误必须显式处理；数据库操作全部参数化防注入
2. **前端代码**: 组件使用 `<script setup lang="ts">`；状态放 Pinia；禁止直接操作 DOM 操作 Three.js 之外的元素
3. **文件命名**: 新文件一律 snake_case（如 `browser_chrome.go`、`bookmark_parser_test.go`）
4. **注释**: 自包含，不引用 md 文档；导出符号必须有 GoDoc 风格注释
5. **敏感数据**: 密码等字段在 DB 中必须为密文（密钥由管理员密码派生），前端仅在管理员解锁后可读取明文
6. **UI 图标**: 禁止 emoji 图标，使用内联 SVG（stroke 风格，24×24，stroke-width 2，currentColor）
7. **主题**: 支持多主题换肤（CSS variables + data-theme），默认暗色科技风

## 常用命令

```bash
wails dev            # 开发模式（热重载）
wails build          # 生产构建
cd frontend && npm run build   # 仅构建前端
go test ./...        # 后端测试
```

## 安全红线

- 管理员密码永不明文落盘；只存 argon2id 哈希 + salt
- 加密密钥用管理员密码经 KDF 派生，派生密钥仅存内存
- 未解锁状态下，API 层拒绝返回明文敏感字段
