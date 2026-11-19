package app

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"my-web-mark/internal/crypto"
	"my-web-mark/internal/store"
)

// 账号类型（Kind）取值。网站账号之外还支持设备/开机密码、WiFi 密码等非网页凭据。
const (
	AccountKindWeb    = "web"
	AccountKindDevice = "device"
	AccountKindWifi   = "wifi"
	AccountKindOther  = "other"
)

// ErrNoPassword 表示该账号条目未保存密码。
var ErrNoPassword = errors.New("app: 该条目没有保存密码")

// AccountService 账号密码服务；明文仅在金库解锁后可见。
type AccountService struct {
	sh *shared
}

// AccountInput 创建/更新账号入参。Password 为明文（本地 IPC 通道），存储前加密；
// 站点名、用户名、密码均可留空，但至少要有其中一项或备注。
type AccountInput struct {
	ThemeID    int64   `json:"themeId"`
	Kind       string  `json:"kind"`
	BookmarkID int64   `json:"bookmarkId"`
	Site       string  `json:"site"`
	Username   string  `json:"username"`
	Password   string  `json:"password"` // 创建/更新传空表示不设置或不改密
	URL        string  `json:"url"`
	Note       string  `json:"note"`
	TagIDs     []int64 `json:"tagIds"`
}

// normalizeKind 归一化账号类型，无法识别时按网站账号处理。
func normalizeKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case AccountKindDevice:
		return AccountKindDevice
	case AccountKindWifi:
		return AccountKindWifi
	case AccountKindOther:
		return AccountKindOther
	default:
		return AccountKindWeb
	}
}

// validate 校验入参；密码非空时是否需要密钥由加密步骤负责。
func (in *AccountInput) validate() error {
	if in.ThemeID <= 0 {
		return fmt.Errorf("app: 必须选择主题")
	}
	if strings.TrimSpace(in.Site) == "" && strings.TrimSpace(in.Username) == "" &&
		strings.TrimSpace(in.Password) == "" && strings.TrimSpace(in.URL) == "" &&
		strings.TrimSpace(in.Note) == "" {
		return fmt.Errorf("app: 站点、用户名、密码、网址、备注至少填写一项")
	}
	return nil
}

// AccountSummary 返回给前端的账号（密码仅掩码；解锁后可通过 reveal 获取明文）。
type AccountSummary struct {
	ID         int64       `json:"id"`
	ThemeID    int64       `json:"themeId"`
	Kind       string      `json:"kind"`
	BookmarkID *int64      `json:"bookmarkId"`
	Site       string      `json:"site"`
	Username   string      `json:"username"`
	HasPwd     bool        `json:"hasPwd"`
	URL        string      `json:"url"`
	Note       string      `json:"note"`
	Tags       []store.Tag `json:"tags"`
}

// toSummary 账号脱敏转换。
func toSummary(a *store.Account) AccountSummary {
	tags := a.Tags
	if tags == nil {
		tags = []store.Tag{}
	}
	return AccountSummary{
		ID: a.ID, ThemeID: a.ThemeID, Kind: a.Kind, BookmarkID: a.BookmarkID,
		Site: a.Site, Username: a.Username, HasPwd: len(a.PasswordCipher) > 0,
		URL: a.URL, Note: a.Note, Tags: tags,
	}
}

// ListByTheme 返回某主题下账号（脱敏，不含明文密码）。
// 金库未解锁也可调用（浏览站点名、用户名、备注、标签不需要解锁）。
func (a *AccountService) ListByTheme(themeID int64) ([]AccountSummary, error) {
	accounts, err := a.sh.store.ListAccountsByTheme(themeID)
	if err != nil {
		return nil, err
	}
	out := make([]AccountSummary, 0, len(accounts))
	for i := range accounts {
		out = append(out, toSummary(&accounts[i]))
	}
	return out, nil
}

// Create 新增账号；只有填写了密码时才要求金库已解锁（加密需要密钥）。
func (a *AccountService) Create(in AccountInput) (*AccountSummary, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	acc := &store.Account{
		ThemeID: in.ThemeID, Kind: normalizeKind(in.Kind), Site: strings.TrimSpace(in.Site),
		Username: strings.TrimSpace(in.Username), URL: strings.TrimSpace(in.URL),
		Note: strings.TrimSpace(in.Note),
	}
	if in.BookmarkID > 0 {
		id := in.BookmarkID
		acc.BookmarkID = &id
	}
	if err := a.fillFromBookmark(acc); err != nil {
		return nil, err
	}
	deriveSite(acc)
	if in.Password != "" {
		cipher, err := a.encryptPassword(in.Password)
		if err != nil {
			return nil, err
		}
		acc.PasswordCipher = cipher
	}
	if err := a.sh.store.CreateAccount(acc); err != nil {
		return nil, err
	}
	if err := a.sh.store.SetAccountTags(acc.ID, a.validTagIDs(acc.ThemeID, in.TagIDs)); err != nil {
		return nil, err
	}
	return a.summaryByID(acc.ID)
}

// Update 更新账号；Password 非空则同时改密（需解锁）。
func (a *AccountService) Update(id int64, in AccountInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	acc := &store.Account{
		ID: id, ThemeID: in.ThemeID, Kind: normalizeKind(in.Kind), Site: strings.TrimSpace(in.Site),
		Username: strings.TrimSpace(in.Username), URL: strings.TrimSpace(in.URL),
		Note: strings.TrimSpace(in.Note),
	}
	if in.BookmarkID > 0 {
		bid := in.BookmarkID
		acc.BookmarkID = &bid
	}
	if err := a.fillFromBookmark(acc); err != nil {
		return err
	}
	deriveSite(acc)
	if in.Password != "" {
		cipher, err := a.encryptPassword(in.Password)
		if err != nil {
			return err
		}
		acc.PasswordCipher = cipher
	}
	if err := a.sh.store.UpdateAccount(acc); err != nil {
		return err
	}
	return a.sh.store.SetAccountTags(id, a.validTagIDs(acc.ThemeID, in.TagIDs))
}

// Delete 删除账号。
func (a *AccountService) Delete(id int64) error {
	return a.sh.store.DeleteAccount(id)
}

// RevealResult 明文密码返回值。
type RevealResult struct {
	ID       int64  `json:"id"`
	Password string `json:"password"`
}

// Reveal 解密返回明文密码；金库锁定时报错。
func (a *AccountService) Reveal(id int64) (*RevealResult, error) {
	acc, err := a.sh.store.GetAccount(id)
	if err != nil {
		return nil, err
	}
	if len(acc.PasswordCipher) == 0 {
		return nil, ErrNoPassword
	}
	var plain []byte
	err = a.sh.withKey(func(key []byte) error {
		var e error
		plain, e = crypto.Decrypt(key, acc.PasswordCipher)
		return e
	})
	if err != nil {
		return nil, err
	}
	return &RevealResult{ID: id, Password: string(plain)}, nil
}

// fillFromBookmark 关联书签时，用书签标题/网址补齐为空的站点名与网址。
func (a *AccountService) fillFromBookmark(acc *store.Account) error {
	if acc.BookmarkID == nil {
		return nil
	}
	bm, err := a.sh.store.GetBookmark(*acc.BookmarkID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			// 书签已删除：静默解除关联，不阻塞保存
			acc.BookmarkID = nil
			return nil
		}
		return err
	}
	if acc.Site == "" {
		acc.Site = bm.Title
	}
	if acc.URL == "" {
		acc.URL = bm.URL
	}
	return nil
}

// deriveSite 站点名为空时尝试用网址域名兜底，便于列表展示。
func deriveSite(acc *store.Account) {
	if acc.Site != "" {
		return
	}
	raw := strings.TrimSpace(acc.URL)
	if raw == "" {
		return
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		acc.Site = strings.TrimPrefix(u.Host, "www.")
	}
}

// validTagIDs 过滤掉不存在或不属于该主题的标签，避免脏关联。
func (a *AccountService) validTagIDs(themeID int64, ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	tags, err := a.sh.store.ListTagsByTheme(themeID)
	if err != nil {
		return nil
	}
	allowed := make(map[int64]bool, len(tags))
	for _, t := range tags {
		allowed[t.ID] = true
	}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if allowed[id] {
			out = append(out, id)
		}
	}
	return out
}

// summaryByID 读回单条账号并转换为脱敏摘要（用于返回新建结果）。
func (a *AccountService) summaryByID(id int64) (*AccountSummary, error) {
	acc, err := a.sh.store.GetAccount(id)
	if err != nil {
		return nil, err
	}
	s := toSummary(acc)
	return &s, nil
}

// encryptPassword 用内存密钥加密明文密码。
func (a *AccountService) encryptPassword(plain string) ([]byte, error) {
	if strings.Contains(plain, "\x00") {
		return nil, fmt.Errorf("app: 密码含非法字符")
	}
	var cipher []byte
	err := a.sh.withKey(func(key []byte) error {
		var e error
		cipher, e = crypto.Encrypt(key, []byte(plain))
		return e
	})
	if err != nil {
		return nil, err
	}
	return cipher, nil
}
