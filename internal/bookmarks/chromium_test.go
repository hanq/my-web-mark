package bookmarks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// makeChromiumFixture 生成临时 Chromium User Data 结构并返回其路径。
func makeChromiumFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	profile := filepath.Join(dir, "Google", "Chrome", "User Data", "Default")
	if err := os.MkdirAll(profile, 0o755); err != nil {
		t.Fatal(err)
	}
	data := map[string]any{
		"roots": map[string]any{
			"bookmark_bar": map[string]any{
				"name": "书签栏", "type": "folder",
				"children": []any{
					map[string]any{"name": "GitHub", "type": "url", "url": "https://github.com"},
					map[string]any{"name": "技术文档", "type": "folder", "children": []any{
						map[string]any{"name": "Go 官网", "type": "url", "url": "https://go.dev"},
						map[string]any{"name": "重复", "type": "url", "url": "https://github.com"},
					}},
				},
			},
			"other": map[string]any{
				"name": "其他书签", "type": "folder",
				"children": []any{
					map[string]any{"name": "MDN", "type": "url", "url": "https://developer.mozilla.org"},
				},
			},
			"synced": map[string]any{"name": "移动设备书签", "type": "folder", "children": []any{}},
		},
	}
	raw, _ := json.Marshal(data)
	if err := os.WriteFile(filepath.Join(profile, "Bookmarks"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestChromeBookmarks(t *testing.T) {
	local := makeChromiumFixture(t)
	roots, err := ChromeBookmarks(local)
	if err != nil {
		t.Fatal(err)
	}
	if len(roots) == 0 {
		t.Fatal("未解析到根节点")
	}
	flat := Flatten(roots...)
	// 期望 3 条（去重后）：GitHub、Go 官网、MDN
	if len(flat) != 3 {
		t.Fatalf("期望 3 条扁平书签，得到 %d: %+v", len(flat), flat)
	}
	folders := map[string]bool{}
	for _, f := range flat {
		folders[f.Folder] = true
	}
	if !folders["书签栏/技术文档"] {
		t.Fatalf("缺少子文件夹路径，实际: %+v", flat)
	}
}

func TestChromeBookmarksNotInstalled(t *testing.T) {
	if _, err := ChromeBookmarks(t.TempDir()); err != ErrNoProfile {
		t.Fatalf("期望 ErrNoProfile，得到 %v", err)
	}
}

func TestFlattenSkipsEmptyAndDup(t *testing.T) {
	roots := []*Node{
		{Name: "空文件夹", Children: nil},
		{Name: "栏", Children: []*Node{
			{Name: "A", URL: "https://a"},
			{Name: "A2", URL: "https://a"}, // 重复 URL
		}},
	}
	flat := Flatten(roots...)
	if len(flat) != 1 || flat[0].URL != "https://a" {
		t.Fatalf("去重失败: %+v", flat)
	}
}
