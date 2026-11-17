package bookmarks

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

// makeFirefoxFixture 生成临时 Firefox Profile（含 places.sqlite）并返回目录。
func makeFirefoxFixture(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	profile := filepath.Join(base, "Profiles", "abc123.default-release")
	if err := os.MkdirAll(profile, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(base, "profiles.ini"), []byte(
		"[Install4F96D1932A9F858E]\nDefault=Profiles/abc123.default-release\n"+
			"[Profile0]\nName=default-release\nIsRelative=1\nPath=Profiles/abc123.default-release\nDefault=1\n"),
		0o644); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", filepath.Join(profile, "places.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	schema := []string{
		`CREATE TABLE moz_places (id INTEGER PRIMARY KEY, url TEXT)`,
		`CREATE TABLE moz_bookmarks (id INTEGER PRIMARY KEY, parent INTEGER, type INTEGER, title TEXT, fk INTEGER)`,
	}
	for _, s := range schema {
		if _, err := db.Exec(s); err != nil {
			t.Fatal(err)
		}
	}
	seed := []string{
		// 根容器（id=1 places root 由 Firefox 创建，这里模拟 toolbar/menu/unfiled 三个）
		`INSERT INTO moz_places (id, url) VALUES
			(10, 'https://github.com'), (11, 'https://go.dev'), (12, 'https://news.ycombinator.com')`,
		`INSERT INTO moz_bookmarks (id, parent, type, title, fk) VALUES
			(1, 0, 2, 'places', NULL)`,
		`INSERT INTO moz_bookmarks (id, parent, type, title, fk) VALUES
			(2, 1, 2, 'toolbar', NULL),
			(3, 1, 2, 'menu', NULL),
			(4, 1, 2, 'unfiled', NULL)`,
		// toolbar 下一个文件夹 + 两个书签
		`INSERT INTO moz_bookmarks (id, parent, type, title, fk) VALUES
			(5, 2, 2, '开发', NULL),
			(6, 5, 1, 'GitHub', 10),
			(7, 5, 1, 'Go', 11),
			(8, 4, 1, 'HN', 12)`,
	}
	for _, s := range seed {
		if _, err := db.Exec(s); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return base
}

func TestFirefoxBookmarks(t *testing.T) {
	appData := makeFirefoxFixture(t)
	roots, err := firefoxBookmarksCommon(appData, filepath.Base(appData)+"/nonexist")
	if err == nil {
		t.Fatal("不存在的目录应返回 ErrNoProfile")
	}

	// 用真实相对路径结构测试
	base := appData
	profileDir := filepath.Join(base, "Profiles", "abc123.default-release")
	places, err := OpenPlacesCopy(profileDir)
	if err != nil {
		t.Fatal(err)
	}
	defer places.Close()

	roots, err = places.BookmarkRoots()
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) == 0 {
		t.Fatal("无根分组")
	}
	flat := Flatten(roots...)
	if len(flat) != 3 {
		t.Fatalf("期望 3 条书签，得到 %d: %+v", len(flat), flat)
	}
	found := map[string]string{}
	for _, f := range flat {
		found[f.URL] = f.Folder
	}
	if found["https://github.com"] != "开发/开发" || found["https://github.com"] == "" {
		// toolbar 根分组名"书签工具栏"下有"开发"文件夹
		if found["https://github.com"] != "书签工具栏/开发" {
			t.Fatalf("GitHub 的路径不符: %+v", found)
		}
	}
	if found["https://news.ycombinator.com"] != "其他书签" {
		t.Fatalf("HN 应在'其他书签': %+v", found)
	}
}

func TestReadProfilesIni(t *testing.T) {
	base := t.TempDir()
	os.WriteFile(filepath.Join(base, "profiles.ini"), []byte(
		"[General]\nStartWithLastProfile=1\n\n[Profile0]\nName=default\nIsRelative=1\nPath=Profiles/xyz.default\nDefault=1\n"), 0o644)
	got := readProfilesIni(base)
	if got != "default" {
		t.Fatalf("期望 default，得到 %q", got)
	}
}
