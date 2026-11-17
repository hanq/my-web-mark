package app

import (
	"path/filepath"
	"testing"

	"my-web-mark/internal/store"
)

// newTestServices 用临时数据库构造全部服务。
func newTestServices(t *testing.T) *Services {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return NewServices(st)
}

func TestFullWorkflow(t *testing.T) {
	s := newTestServices(t)

	// 1. 初始状态：需要设置管理员密码
	if st := s.Vault.Status(); !st.SetupRequired {
		t.Fatalf("初始状态应为 SetupRequired: %+v", st)
	}

	// 2. 创建主题
	th, err := s.Theme.Create(ThemeInput{Name: "开发", Color: "#4f8cff"})
	if err != nil {
		t.Fatal(err)
	}
	th2, err := s.Theme.Create(ThemeInput{Name: "设计", Color: "#ff6b6b"})
	if err != nil {
		t.Fatal(err)
	}

	// 3. 未解锁时创建账号应失败
	_, err = s.Account.Create(AccountInput{
		ThemeID: th.ID, Site: "GitHub", Username: "me", Password: "p@ss",
	})
	if err != ErrVaultLocked {
		t.Fatalf("未解锁创建账号应返回 ErrVaultLocked，得到 %v", err)
	}

	// 4. 设置管理员密码（自动解锁）
	if err := s.Vault.Setup("admin123"); err != nil {
		t.Fatal(err)
	}
	if st := s.Vault.Status(); !st.Unlocked {
		t.Fatal("设置后应已解锁")
	}

	// 5. 创建账号
	acc, err := s.Account.Create(AccountInput{
		ThemeID: th.ID, Site: "GitHub", Username: "me@example.com", Password: "s3cret!",
	})
	if err != nil {
		t.Fatal(err)
	}

	// 6. 列表应脱敏（无明文）
	list, err := s.Account.ListByTheme(th.ID)
	if err != nil || len(list) != 1 {
		t.Fatalf("账号列表异常: %v %+v", err, list)
	}

	// 7. 解锁状态下 Reveal 明文正确
	rev, err := s.Account.Reveal(acc.ID)
	if err != nil || rev.Password != "s3cret!" {
		t.Fatalf("Reveal 结果错误: %v %+v", err, rev)
	}

	// 8. 锁定后 Reveal 失败
	s.Vault.Lock()
	if _, err := s.Account.Reveal(acc.ID); err != ErrVaultLocked {
		t.Fatalf("锁定后 Reveal 应返回 ErrVaultLocked，得到 %v", err)
	}

	// 9. 重新解锁并修改管理员密码
	if err := s.Vault.Unlock(UnlockInput{Password: "admin123"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Vault.ChangePassword(ChangePasswordInput{
		OldPassword: "admin123", NewPassword: "newpwd456",
	}); err != nil {
		t.Fatal(err)
	}

	// 10. 改密后旧密文可被新密钥解开
	rev, err = s.Account.Reveal(acc.ID)
	if err != nil || rev.Password != "s3cret!" {
		t.Fatalf("改密后 Reveal 失败: %v %+v", err, rev)
	}

	// 11. 锁定后用新密码解锁
	s.Vault.Lock()
	if err := s.Vault.Unlock(UnlockInput{Password: "newpwd456"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Vault.Unlock(UnlockInput{Password: "newpwd456"}); err != ErrVaultAlreadyUnlocked {
		t.Fatalf("重复解锁应返回 ErrVaultAlreadyUnlocked，得到 %v", err)
	}

	// 12. 书签 CRUD
	bm, err := s.Bookmark.Create(BookmarkInput{
		ThemeID: th.ID, Title: "Go", URL: "go.dev",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bm.URL != "https://go.dev" {
		t.Fatalf("URL 未自动补协议: %s", bm.URL)
	}
	// 重复 URL 应报已存在
	if _, err := s.Bookmark.Create(BookmarkInput{ThemeID: th.ID, Title: "Go2", URL: "https://go.dev"}); err == nil {
		t.Fatal("重复书签未拦截")
	}
	if err := s.Bookmark.Update(bm.ID, BookmarkInput{
		ThemeID: th2.ID, Title: "Go 官网", URL: "https://go.dev", Pinned: true,
	}); err != nil {
		t.Fatal(err)
	}
	got, err := s.Bookmark.ListByTheme(th2.ID)
	if err != nil || len(got) != 1 || !got[0].Pinned {
		t.Fatalf("更新/移动书签失败: %v %+v", err, got)
	}

	// 13. 删除主题级联
	if err := s.Theme.Delete(th.ID); err != nil {
		t.Fatal(err)
	}
	list2, err := s.Account.ListByTheme(th.ID)
	if err != nil || len(list2) != 0 {
		t.Fatalf("级联删除账号失败: %v %+v", err, list2)
	}
}

func TestParseDroppedURL(t *testing.T) {
	s := newTestServices(t)
	b := s.Browser

	cases := []struct {
		in    string
		want  string
		isErr bool
	}{
		{"https://github.com/foo", "https://github.com/foo", false},
		{"http://a.com\nhttps://b.com", "http://a.com", false},
		{"# comment\nhttps://real.com/x", "https://real.com/x", false},
		{`<a href="https://link.com/page">拖拽</a>`, "https://link.com/page", false},
		{"example.com/path", "https://example.com/path", false},
		{"随便一段文字", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, err := b.ParseDroppedURL(c.in)
		if c.isErr {
			if err == nil {
				t.Fatalf("输入 %q 应报错", c.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("输入 %q 报错: %v", c.in, err)
		}
		if got.RawURL != c.want {
			t.Fatalf("输入 %q: 期望 %q 得到 %q", c.in, c.want, got.RawURL)
		}
	}
}
