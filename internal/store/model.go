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

// Bookmark 书签条目；Tags 为其标签（查询时聚合填充）。
type Bookmark struct {
	ID      int64  `json:"id"`
	ThemeID int64  `json:"themeId"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Favicon string `json:"favicon"`
	Note    string `json:"note"`
	Pinned  bool   `json:"pinned"`
	Source  string `json:"source"` // manual/edge/chrome/zen/firefox
	Tags    []Tag  `json:"tags"`
	CreatedAt time.Time `json:"createdAt"`
}

// Account 账号条目（网站账号、设备/开机密码、WiFi 密码等）。
// Kind 区分类型：web/device/wifi/other；BookmarkID 可关联同主题的书签，
// 用于自动带出站点名与网址；PasswordCipher 为密文，明文仅在金库解锁后由服务层解出。
type Account struct {
	ID             int64     `json:"id"`
	ThemeID        int64     `json:"themeId"`
	Kind           string    `json:"kind"`
	BookmarkID     *int64    `json:"bookmarkId"`
	Site           string    `json:"site"`
	Username       string    `json:"username"`
	PasswordCipher []byte    `json:"-"`
	URL            string    `json:"url"`
	Note           string    `json:"note"`
	Tags           []Tag     `json:"tags"`
	CreatedAt      time.Time `json:"createdAt"`
}

// VaultMeta 管理员凭据元数据（单行表）。
type VaultMeta struct {
	Salt         []byte    `json:"-"`
	PasswordHash []byte    `json:"-"`
	KDFParams    string    `json:"kdfParams"`
	CreatedAt    time.Time `json:"createdAt"`
}
