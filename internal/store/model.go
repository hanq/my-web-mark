package store

import "time"

// Theme 主题分组，对应 2.5D 场景中的一个"岛屿"。
type Theme struct {
	ID    int64     `json:"id"`
	Name  string    `json:"name"`
	Color string    `json:"color"`
	Icon  string    `json:"icon"`
	Sort  int       `json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
}

// Bookmark 书签条目。
type Bookmark struct {
	ID      int64  `json:"id"`
	ThemeID int64  `json:"themeId"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Favicon string `json:"favicon"`
	Note    string `json:"note"`
	Pinned  bool   `json:"pinned"`
	Source  string `json:"source"` // manual/edge/chrome/zen/firefox
	CreatedAt time.Time `json:"createdAt"`
}

// Account 账号条目；PasswordCipher 为密文，明文仅在金库解锁后由服务层解出。
type Account struct {
	ID             int64     `json:"id"`
	ThemeID        int64     `json:"themeId"`
	Site           string    `json:"site"`
	Username       string    `json:"username"`
	PasswordCipher []byte    `json:"-"`
	URL            string    `json:"url"`
	Note           string    `json:"note"`
	CreatedAt      time.Time `json:"createdAt"`
}

// VaultMeta 管理员凭据元数据（单行表）。
type VaultMeta struct {
	Salt         []byte    `json:"-"`
	PasswordHash []byte    `json:"-"`
	KDFParams    string    `json:"kdfParams"`
	CreatedAt    time.Time `json:"createdAt"`
}
