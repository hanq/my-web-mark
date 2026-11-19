// Package app 提供 Wails 绑定层：前端通过 window.go.app.* 调用这些服务。
// 各服务共享同一 Store 与内存态（金库解锁密钥）。
package app

import (
	"context"
	"errors"
	"sync"

	"my-web-mark/internal/favicon"
	"my-web-mark/internal/store"
)

// ErrVaultLocked 金库未解锁时访问敏感操作的统一错误。
var ErrVaultLocked = errors.New("app: 金库未解锁，请先输入管理员密码")

// ErrVaultSetupRequired 尚未设置管理员密码。
var ErrVaultSetupRequired = errors.New("app: 尚未设置管理员密码")

// ErrVaultAlreadySetup 管理员密码已设置，不可重复初始化。
var ErrVaultAlreadySetup = errors.New("app: 管理员密码已设置")

// ErrVaultAlreadyUnlocked 金库已处于解锁状态。
var ErrVaultAlreadyUnlocked = errors.New("app: 金库已解锁")

// shared 服务间共享的运行时状态。
type shared struct {
	store *store.Store

	// ctx 为 Wails 运行时上下文（OnStartup 注入），对话框等运行时能力依赖它
	ctx context.Context

	mu       sync.RWMutex // 保护 key 与 unlocked
	aesKey   []byte       // 解锁后的 AES-256 主密钥（锁定时为 nil 并清零）
	unlocked bool
}

// withKey 在持锁状态下执行需要密钥的操作。
func (s *shared) withKey(fn func(key []byte) error) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.unlocked || s.aesKey == nil {
		return ErrVaultLocked
	}
	return fn(s.aesKey)
}

// Services 汇总全部绑定服务；Wails Bind 使用其字段。
type Services struct {
	Vault     *VaultService
	Theme     *ThemeService
	Tag       *TagService
	Bookmark  *BookmarkService
	Account   *AccountService
	Browser   *BrowserService
	PageTitle *PageTitleService
	Favicon   *FaviconService
	Settings  *SettingsService
	Export    *ExportService

	sh *shared
}

// NewServices 用已打开的 Store 与 favicon 缓存构造全部服务。
func NewServices(st *store.Store, favCache *favicon.Cache) *Services {
	sh := &shared{store: st}
	return &Services{
		Vault:     &VaultService{sh: sh},
		Theme:     &ThemeService{sh: sh},
		Tag:       &TagService{sh: sh},
		Bookmark:  &BookmarkService{sh: sh},
		Account:   &AccountService{sh: sh},
		Browser:   &BrowserService{sh: sh},
		PageTitle: &PageTitleService{},
		Favicon:   &FaviconService{cache: favCache},
		Settings:  &SettingsService{sh: sh},
		Export:    &ExportService{sh: sh},
		sh:        sh,
	}
}

// SetContext 注入 Wails 运行时上下文（OnStartup 时调用）。
func (s *Services) SetContext(ctx context.Context) {
	s.sh.ctx = ctx
}

// BindTargets 返回 Wails Bind 所需的服务实例切片。
func (s *Services) BindTargets() []interface{} {
	return []interface{}{s.Vault, s.Theme, s.Tag, s.Bookmark, s.Account, s.Browser, s.PageTitle, s.Favicon, s.Settings, s.Export}
}

// Shutdown 释放资源（关闭数据库、清零密钥）。
func (s *Services) Shutdown() {
	s.Vault.Lock()
	s.sh.store.Close()
}
