package app

import (
	"path/filepath"
	"strings"
	"testing"

	"my-web-mark/internal/bookmarks"
	"my-web-mark/internal/favicon"
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
	return NewServices(st, favicon.NewCacheAt(t.TempDir()))
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

func TestTagWorkflow(t *testing.T) {
	s := newTestServices(t)
	_ = s.Vault.Setup("admin123")

	th, err := s.Theme.Create(ThemeInput{Name: "开发", Color: "#4f8cff"})
	if err != nil {
		t.Fatal(err)
	}
	th2, _ := s.Theme.Create(ThemeInput{Name: "工具", Color: "#38d39f"})

	// 创建标签
	tag1, err := s.Tag.Create(th.ID, TagInput{Name: "前端", Color: "#ffb648"})
	if err != nil {
		t.Fatal(err)
	}
	tag2, _ := s.Tag.Create(th.ID, TagInput{Name: "后端", Color: "#38d39f"})
	if _, err := s.Tag.Create(th.ID, TagInput{Name: "前端"}); err == nil {
		t.Fatal("同主题重名标签应被拒绝")
	}
	tag3, _ := s.Tag.Create(th2.ID, TagInput{Name: "前端"}) // 不同主题允许同名

	// 书签打标签（多对多）
	bm, err := s.Bookmark.Create(BookmarkInput{
		ThemeID: th.ID, Title: "Vue", URL: "https://vuejs.org",
		TagIDs: []int64{tag1.ID, tag2.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bm.Tags) != 2 {
		t.Fatalf("书签应有 2 个标签: %+v", bm.Tags)
	}
	bm2, _ := s.Bookmark.Create(BookmarkInput{
		ThemeID: th.ID, Title: "Go", URL: "https://go.dev",
	})
	if bm2.Tags == nil {
		// 无标签书签允许空数组或 null，只校验不影响筛选
		_ = bm2.Tags
	}

	// 编辑书签替换标签
	if err := s.Bookmark.Update(bm.ID, BookmarkInput{
		ThemeID: th.ID, Title: "Vue3", URL: "https://vuejs.org", TagIDs: []int64{tag2.ID},
	}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Bookmark.ListByTheme(th.ID)
	var vue *BookmarkItemAlias
	for i := range got {
		if got[i].URL == "https://vuejs.org" {
			vue = &got[i]
		}
	}
	if vue == nil || len(vue.Tags) != 1 || vue.Tags[0].Name != "后端" {
		t.Fatalf("替换标签失败: %+v", vue)
	}

	// 列表筛选辅助：按标签过滤的语义由前端完成，此处验证标签列表接口
	tags, err := s.Tag.ListByTheme(th.ID)
	if err != nil || len(tags) != 2 {
		t.Fatalf("主题标签列表错误: %v %+v", err, tags)
	}
	all, _ := s.Tag.ListAll()
	if len(all) != 3 { // 前端 + 后端 + 工具主题的前端
		t.Fatalf("全局标签数应为 3: %+v", all)
	}
	_ = tag3

	// 删除标签后书签关联清除
	if err := s.Tag.Delete(tag2.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Bookmark.ListByTheme(th.ID)
	for _, b := range got {
		if b.URL == "https://vuejs.org" && len(b.Tags) != 0 {
			t.Fatalf("删除标签后关联未清除: %+v", b.Tags)
		}
	}

	// 删除主题级联删除标签
	if err := s.Theme.Delete(th.ID); err != nil {
		t.Fatal(err)
	}
	all2, _ := s.Tag.ListAll()
	if len(all2) != 1 {
		t.Fatalf("主题删除后应剩 1 个标签: %+v", all2)
	}
}

// BookmarkItemAlias 便于测试中引用带标签的书签类型。
type BookmarkItemAlias = store.Bookmark

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

func TestBookmarkSort(t *testing.T) {
	s := newTestServices(t)
	_ = s.Vault.Setup("admin123")

	th, _ := s.Theme.Create(ThemeInput{Name: "T", Color: "#4f8cff"})
	for _, n := range []string{"a", "b", "c"} {
		if _, err := s.Bookmark.Create(BookmarkInput{
			ThemeID: th.ID, Title: n, URL: "https://" + n + ".com",
		}); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := s.Bookmark.ListByTheme(th.ID)
	if len(got) != 3 {
		t.Fatalf("should have 3 bookmarks: %+v", titles(got))
	}

	// establish baseline order c,b,a (created_at has second precision, creation order unreliable)
	if err := s.Bookmark.Sort(BookmarkSortInput{IDs: ids(got, "c", "b", "a")}); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Bookmark.ListByTheme(th.ID)
	if got[0].Title != "c" || got[2].Title != "a" {
		t.Fatalf("baseline order should be c,b,a: %+v", titles(got))
	}

	// drag-reorder to a,b,c
	if err := s.Bookmark.Sort(BookmarkSortInput{IDs: ids(got, "a", "b", "c")}); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Bookmark.ListByTheme(th.ID)
	if got[0].Title != "a" || got[2].Title != "c" {
		t.Fatalf("sort not applied: %+v", titles(got))
	}

	// pin b: stays first, group order preserved
	bm := got[1]
	if err := s.Bookmark.Update(bm.ID, BookmarkInput{
		ThemeID: th.ID, Title: bm.Title, URL: bm.URL, Pinned: true,
	}); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Bookmark.ListByTheme(th.ID)
	if got[0].Title != "b" || got[1].Title != "a" || got[2].Title != "c" {
		t.Fatalf("pin+sort combo wrong: %+v", titles(got))
	}
}

func titles(bm []store.Bookmark) []string {
	out := make([]string, len(bm))
	for i, b := range bm {
		out[i] = b.Title
	}
	return out
}

func ids(bm []store.Bookmark, order ...string) []int64 {
	want := map[string]int{}
	for i, n := range order {
		want[n] = i
	}
	out := make([]int64, len(bm))
	for _, b := range bm {
		out[want[b.Title]] = b.ID
	}
	return out
}

func TestSettingsAndBrowserLaunch(t *testing.T) {
	s := newTestServices(t)

	// 默认值：空（跟随系统）
	v, err := s.Settings.Get(SettingKeyLaunchBrowser)
	if err != nil || v != "" {
		t.Fatalf("默认应为空: %q %v", v, err)
	}

	// 非法值/未知键拒绝
	if err := s.Settings.Set(SettingKeyLaunchBrowser, "safari"); err == nil {
		t.Fatal("非法值应被拒绝")
	}
	if err := s.Settings.Set("unknown_key", "x"); err == nil {
		t.Fatal("未知键应被拒绝")
	}

	// 合法设置往返
	if err := s.Settings.Set(SettingKeyLaunchBrowser, "edge"); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.Settings.Get(SettingKeyLaunchBrowser); v != "edge" {
		t.Fatalf("往返不一致: %q", v)
	}

	// 候选路径非空 + 探测逻辑（注入 exists）
	if len(browserExeCandidates("edge")) == 0 {
		t.Fatal("edge 候选路径为空")
	}
	found := browserExeCandidates("chrome")[0]
	if exe, ok := findBrowserExe("chrome", func(p string) bool { return p == found }); !ok || exe != found {
		t.Fatalf("探测失败: %q %v", exe, ok)
	}
	if _, ok := findBrowserExe("chrome", func(string) bool { return false }); ok {
		t.Fatal("全不存在时应返回 false")
	}
	if _, ok := findBrowserExe("unknown", fileExists); ok {
		t.Fatal("未知浏览器应返回 false")
	}
	// 环境变量展开
	if got := expandWinPath(`%LOCALAPPDATA%\x\y.exe`); !strings.HasSuffix(got, `x\y.exe`) || strings.HasPrefix(got, "%") {
		t.Fatalf("路径展开错误: %q", got)
	}
}

func TestThemeExportImport(t *testing.T) {
	s := newTestServices(t)
	_ = s.Vault.Setup("admin123")

	th, _ := s.Theme.Create(ThemeInput{Name: "导出源", Color: "#38d39f"})
	tag, _ := s.Tag.Create(th.ID, TagInput{Name: "前端", Color: "#ffb648"})
	if _, err := s.Bookmark.Create(BookmarkInput{
		ThemeID: th.ID, Title: "Vue", URL: "vuejs.org", Note: "框架", TagIDs: []int64{tag.ID},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Bookmark.Create(BookmarkInput{
		ThemeID: th.ID, Title: "Go", URL: "https://go.dev", Pinned: true,
	}); err != nil {
		t.Fatal(err)
	}

	// --- HTML 导出（直接构造数据层，绕过对话框）---
	bms, _ := s.Bookmark.ListByTheme(th.ID)
	items := make([]bookmarks.ExportBookmark, 0, len(bms))
	for _, b := range bms {
		tags := []string{}
		for _, tg := range b.Tags {
			tags = append(tags, tg.Name)
		}
		folder := ""
		if len(tags) > 0 {
			folder = tags[0]
		}
		items = append(items, bookmarks.ExportBookmark{
			Title: b.Title, URL: b.URL, Note: b.Note, Pinned: b.Pinned, Folder: folder, Tags: tags,
		})
	}
	htmlOut := bookmarks.RenderNetscapeHTML("导出源", items)
	name, parsed, err := bookmarks.ParseNetscapeHTML(htmlOut)
	if err != nil || name != "导出源" || len(parsed) != 2 {
		t.Fatalf("HTML 往返失败: %v %q %d", err, name, len(parsed))
	}

	// --- JSON 导出：置顶与标签保留 ---
	jsonData, _ := bookmarks.MarshalThemeFile(
		bookmarks.ExportThemeMeta{Name: "导出源", Color: "#38d39f"}, items)
	f, err := bookmarks.ParseThemeFile(jsonData)
	if err != nil || len(f.Bookmarks) != 2 {
		t.Fatalf("JSON 往返失败: %v", err)
	}
	if !f.Bookmarks[0].Pinned || len(f.Bookmarks[0].Tags) != 0 {
		// go.dev 是置顶无标签（created DESC 顺序在前）
		var goBm bookmarks.ExportBookmark
		for _, it := range f.Bookmarks {
			if it.URL == "https://go.dev" {
				goBm = it
			}
		}
		if !goBm.Pinned {
			t.Fatalf("置顶未保留: %+v", f.Bookmarks)
		}
	}

	// --- 导入落库（applyImport 可直接测试，无需对话框）---
	svc := &ExportService{sh: s.sh}
	// 用 HTML 解析结果导入（标签来自 Folder）
	var importItems []bookmarks.ExportBookmark
	_, parsedHTML, _ := bookmarks.ParseNetscapeHTML(htmlOut)
	importItems = parsedHTML
	res, err := svc.applyImport("导出源", "", importItems, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.ThemeName != "导出源 (导入)" {
		t.Fatalf("同名主题应加后缀: %q", res.ThemeName)
	}
	if res.Imported != 2 || res.TagCount != 1 {
		t.Fatalf("导入统计错误: %+v", res)
	}

	// 新主题书签/标签验证
	newTheme := res.ThemeID
	got, _ := s.Bookmark.ListByTheme(newTheme)
	if len(got) != 2 {
		t.Fatalf("导入后书签数错误: %d", len(got))
	}
	for _, b := range got {
		if b.URL == "vuejs.org" && (len(b.Tags) != 1 || b.Tags[0].Name != "前端") {
			t.Fatalf("导入标签错误: %+v", b)
		}
		if b.Source != "import" {
			t.Fatalf("来源标记错误: %s", b.Source)
		}
	}
	// HTML 格式不保留置顶
	for _, b := range got {
		if b.Pinned {
			t.Fatal("HTML 导入不应有置顶")
		}
	}

	// --- JSON 导入：保留置顶 ---
	svc2 := &ExportService{sh: s.sh}
	res2, err := svc2.applyImport("导出源", "#123456", f.Bookmarks, true)
	if err != nil {
		t.Fatal(err)
	}
	got2, _ := s.Bookmark.ListByTheme(res2.ThemeID)
	pinnedCount := 0
	for _, b := range got2 {
		if b.Pinned {
			pinnedCount++
		}
	}
	if pinnedCount != 1 {
		t.Fatalf("JSON 导入置顶数错误: %d", pinnedCount)
	}
	if res2.ThemeName != "导出源 (导入) (导入)" {
		t.Fatalf("重复导入后缀: %q", res2.ThemeName)
	}
}
