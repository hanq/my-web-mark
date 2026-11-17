package app

import (
	"fmt"
	"strings"

	"my-web-mark/internal/crypto"
	"my-web-mark/internal/store"
)

// AccountService 账号密码服务；明文仅在金库解锁后可见。
type AccountService struct {
	sh *shared
}

// AccountInput 创建/更新账号入参。Password 为明文（HTTPS 之外的本地 IPC 传输，
// Wails 本地通道），存储前加密。
type AccountInput struct {
	ThemeID  int64  `json:"themeId"`
	Site     string `json:"site"`
	Username string `json:"username"`
	Password string `json:"password"` // 创建必填；更新传空表示不改密
	URL      string `json:"url"`
	Note     string `json:"note"`
}

func (in *AccountInput) validate(requirePwd bool) error {
	if in.ThemeID <= 0 {
		return fmt.Errorf("app: 必须选择主题")
	}
	if in.Site == "" {
		return fmt.Errorf("app: 站点名称不能为空")
	}
	if in.Username == "" {
		return fmt.Errorf("app: 用户名不能为空")
	}
	if requirePwd && in.Password == "" {
		return fmt.Errorf("app: 密码不能为空")
	}
	return nil
}

// AccountSummary 返回给前端的账号（密码仅掩码；解锁后可通过 reveal 获取明文）。
type AccountSummary struct {
	ID       int64  `json:"id"`
	ThemeID  int64  `json:"themeId"`
	Site     string `json:"site"`
	Username string `json:"username"`
	HasPwd   bool   `json:"hasPwd"`
	URL      string `json:"url"`
	Note     string `json:"note"`
}

// toSummary 账号脱敏转换。
func toSummary(a *store.Account) AccountSummary {
	return AccountSummary{
		ID: a.ID, ThemeID: a.ThemeID, Site: a.Site,
		Username: a.Username, HasPwd: len(a.PasswordCipher) > 0,
		URL: a.URL, Note: a.Note,
	}
}

// ListByTheme 返回某主题下账号（脱敏，不含明文密码）。
// 金库未解锁也可调用（显示掩码列表）。
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

// Create 新增账号；要求金库已解锁（加密需要密钥）。
func (a *AccountService) Create(in AccountInput) (*AccountSummary, error) {
	if err := in.validate(true); err != nil {
		return nil, err
	}
	cipher, err := a.encryptPassword(in.Password)
	if err != nil {
		return nil, err
	}
	acc := &store.Account{
		ThemeID: in.ThemeID, Site: in.Site, Username: in.Username,
		PasswordCipher: cipher, URL: in.URL, Note: in.Note,
	}
	if err := a.sh.store.CreateAccount(acc); err != nil {
		return nil, err
	}
	s := toSummary(acc)
	return &s, nil
}

// Update 更新账号；Password 非空则同时改密（需解锁）。
func (a *AccountService) Update(id int64, in AccountInput) error {
	if err := in.validate(false); err != nil {
		return err
	}
	acc := &store.Account{
		ID: id, ThemeID: in.ThemeID, Site: in.Site, Username: in.Username,
		URL: in.URL, Note: in.Note,
	}
	if in.Password != "" {
		cipher, err := a.encryptPassword(in.Password)
		if err != nil {
			return err
		}
		acc.PasswordCipher = cipher
	}
	return a.sh.store.UpdateAccount(acc)
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
