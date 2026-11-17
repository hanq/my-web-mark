package bookmarks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// chromiumBookmarkFile Chromium 系浏览器的 Bookmarks JSON 文件结构（只取需要的字段）。
type chromiumBookmarkFile struct {
	Roots struct {
		BookmarkBar *chromiumNode `json:"bookmark_bar"`
		Other       *chromiumNode `json:"other"`
		Synced      *chromiumNode `json:"synced"`
	} `json:"roots"`
}

type chromiumNode struct {
	Name     string          `json:"name"`
	Type     string          `json:"type"` // "url" 或 "folder"
	URL      string          `json:"url"`
	Children []*chromiumNode `json:"children"`
}

// chromeDataDir 返回 Chromium 系浏览器 User Data 目录（不存在时报 ErrNoProfile）。
func chromeDataDir(localAppData, dirName string) (string, error) {
	dir := filepath.Join(localAppData, dirName)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return "", ErrNoProfile
	}
	return dir, nil
}

// parseChromiumBookmarks 解析 Chromium 系 Bookmarks JSON 文件。
// 返回书签栏 / 其他书签 / 移动设备书签三个根节点。
func parseChromiumBookmarks(path string) (bar, other, synced *Node, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("bookmarks: 读取 %s 失败: %w", path, err)
	}
	var f chromiumBookmarkFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, nil, nil, fmt.Errorf("bookmarks: 解析 %s 失败: %w", path, err)
	}
	return convertChromium(f.Roots.BookmarkBar), convertChromium(f.Roots.Other), convertChromium(f.Roots.Synced), nil
}

// convertChromium 递归转换 Chromium 节点树；URL 型节点才保留地址。
func convertChromium(n *chromiumNode) *Node {
	if n == nil {
		return nil
	}
	node := &Node{Name: n.Name}
	if n.Type == "url" {
		node.URL = n.URL
		return node
	}
	for _, c := range n.Children {
		if child := convertChromium(c); child != nil {
			node.Children = append(node.Children, child)
		}
	}
	return node
}

// ChromiumBookmarks 按 profileName 解析 Chromium 系浏览器书签。
// profileName 为 "Default" 或 "Profile 1" 等；空则自动选第一个可用 Profile。
func ChromiumBookmarks(localAppData, dirName, profileName string) ([]*Node, error) {
	dataDir, err := chromeDataDir(localAppData, dirName)
	if err != nil {
		return nil, err
	}
	if profileName == "" {
		profileName, err = firstChromiumProfile(dataDir)
		if err != nil {
			return nil, err
		}
	}
	bar, other, synced, err := parseChromiumBookmarks(
		filepath.Join(dataDir, profileName, "Bookmarks"))
	if err != nil {
		return nil, err
	}
	var roots []*Node
	for _, n := range []*Node{bar, other, synced} {
		if n != nil {
			roots = append(roots, n)
		}
	}
	return roots, nil
}

// firstChromiumProfile 返回第一个含有 Bookmarks 文件的 Profile 目录名。
func firstChromiumProfile(dataDir string) (string, error) {
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return "", ErrNoProfile
	}
	candidates := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "Default" || name == "Profile 1" {
			candidates = append([]string{name}, candidates...) // 优先常用 Profile
			continue
		}
		if len(name) > 8 && name[:8] == "Profile " {
			candidates = append(candidates, name)
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(dataDir, c, "Bookmarks")); err == nil {
			return c, nil
		}
	}
	return "", ErrNoProfile
}

// EdgeBookmarks 解析 Edge 书签。
func EdgeBookmarks(localAppData string) ([]*Node, error) {
	return ChromiumBookmarks(localAppData, filepath.Join("Microsoft", "Edge", "User Data"), "")
}

// ChromeBookmarks 解析 Chrome 书签。
func ChromeBookmarks(localAppData string) ([]*Node, error) {
	return ChromiumBookmarks(localAppData, filepath.Join("Google", "Chrome", "User Data"), "")
}
