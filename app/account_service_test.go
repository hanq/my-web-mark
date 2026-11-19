package app

import (
	"strings"
	"testing"
)

// TestAccountWithoutPasswordAndLink 校验无需解锁即可新增无密码条目、
// 关联书签自动带出名称与网址、标签关联与筛选过滤。
func TestAccountWithoutPasswordAndLink(t *testing.T) {
	s := newTestServices(t)

	th, err := s.Theme.Create(ThemeInput{Name: "家里", Color: "#4f8cff"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.Theme.Create(ThemeInput{Name: "公司", Color: "#ff6b6b"})
	if err != nil {
		t.Fatal(err)
	}
	tagHome, err := s.Tag.Create(th.ID, TagInput{Name: "WiFi", Color: "#22c55e"})
	if err != nil {
		t.Fatal(err)
	}
	foreignTag, err := s.Tag.Create(other.ID, TagInput{Name: "其他主题", Color: "#888888"})
	if err != nil {
		t.Fatal(err)
	}

	// 未解锁也能新增不涉密的条目（如 WiFi 名称+密码留空）
	wifi, err := s.Account.Create(AccountInput{
		ThemeID: th.ID, Kind: AccountKindWifi, Site: "Home-5G", Note: "客厅路由器",
		TagIDs: []int64{tagHome.ID, foreignTag.ID},
	})
	if err != nil {
		t.Fatalf("未解锁新增无密码条目应成功，得到 %v", err)
	}
	if wifi.Kind != AccountKindWifi || wifi.HasPwd {
		t.Fatalf("类型/密码标记不符合预期: %+v", wifi)
	}
	if len(wifi.Tags) != 1 || wifi.Tags[0].ID != tagHome.ID {
		t.Fatalf("应只保留本主题标签，实际 %+v", wifi.Tags)
	}

	// 无密码条目 Reveal 应报 ErrNoPassword
	if _, err := s.Account.Reveal(wifi.ID); err != ErrNoPassword {
		t.Fatalf("无密码条目 Reveal 应返回 ErrNoPassword，得到 %v", err)
	}

	// 关联书签：站点名/网址留空时自动从书签补齐
	bm, err := s.Bookmark.Create(BookmarkInput{
		ThemeID: th.ID, Title: "路由器管理页", URL: "https://192.168.1.1/admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	linked, err := s.Account.Create(AccountInput{
		ThemeID: th.ID, Kind: AccountKindWeb, BookmarkID: bm.ID, Username: "admin",
	})
	if err != nil {
		t.Fatal(err)
	}
	if linked.Site != "路由器管理页" || linked.URL != "https://192.168.1.1/admin" {
		t.Fatalf("关联书签应补齐站点与网址: %+v", linked)
	}
	if linked.BookmarkID == nil || *linked.BookmarkID != bm.ID {
		t.Fatalf("bookmarkId 未保存: %+v", linked)
	}

	// 站点名留空但有网址：用域名兜底展示
	derived, err := s.Account.Create(AccountInput{
		ThemeID: th.ID, URL: "https://www.example.com/login", Username: "u1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if derived.Site != "example.com" {
		t.Fatalf("站点名应由域名推导，实际 %q", derived.Site)
	}

	// 所有字段都空应被拒绝
	if _, err := s.Account.Create(AccountInput{ThemeID: th.ID}); err == nil ||
		!strings.Contains(err.Error(), "至少填写一项") {
		t.Fatalf("全空条目应报错，得到 %v", err)
	}

	// 列表按主题返回，且包含标签
	list, err := s.Account.ListByTheme(th.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 {
		t.Fatalf("应返回 3 条账号，实际 %d", len(list))
	}

	// 设置管理员密码后可保存密码，且可解密
	if err := s.Vault.Setup("admin123"); err != nil {
		t.Fatal(err)
	}
	upd := AccountInput{ThemeID: th.ID, Kind: AccountKindWifi, Site: "Home-5G", Password: "wifi-pass"}
	if err := s.Account.Update(wifi.ID, upd); err != nil {
		t.Fatal(err)
	}
	res, err := s.Account.Reveal(wifi.ID)
	if err != nil {
		t.Fatal(err)
	}
	if res.Password != "wifi-pass" {
		t.Fatalf("解密结果不符: %q", res.Password)
	}

	// 书签删除后，账号关联被置空（ON DELETE SET NULL）
	if err := s.Bookmark.Delete(bm.ID); err != nil {
		t.Fatal(err)
	}
	acc, err := s.Account.summaryByID(linked.ID)
	if err != nil {
		t.Fatal(err)
	}
	if acc.BookmarkID != nil {
		t.Fatalf("书签删除后关联应清空，实际 %+v", acc.BookmarkID)
	}
}
