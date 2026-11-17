package bookmarks

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// firefoxProfileDir 定位 Firefox 系（Firefox/Zen）的 Profile 目录。
// rootsDir 为 %APPDATA% 下的产品目录，如 "Mozilla/Firefox" 或 "zen-browser-data"。
func firefoxProfileDir(appData, rootsDir string) (string, error) {
	base := filepath.Join(appData, rootsDir)
	if st, err := os.Stat(base); err != nil || !st.IsDir() {
		return "", ErrNoProfile
	}
	// 优先读 profiles.ini 中 Default=1 的 Profile
	if ini := readProfilesIni(base); ini != "" {
		if _, err := os.Stat(filepath.Join(base, ini, "places.sqlite")); err == nil {
			return filepath.Join(base, ini), nil
		}
	}
	// 兜底：扫描含 places.sqlite 的目录，default-release 优先
	entries, err := os.ReadDir(base)
	if err != nil {
		return "", ErrNoProfile
	}
	fallback := ""
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		places := filepath.Join(base, e.Name(), "places.sqlite")
		if _, err := os.Stat(places); err != nil {
			continue
		}
		if fallback == "" || strings.Contains(e.Name(), "default-release") {
			fallback = filepath.Join(base, e.Name())
		}
	}
	if fallback != "" {
		return fallback, nil
	}
	return "", ErrNoProfile
}

// readProfilesIni 解析 profiles.ini 取默认 Profile 的相对目录名；失败返回空串。
func readProfilesIni(base string) string {
	raw, err := os.ReadFile(filepath.Join(base, "profiles.ini"))
	if err != nil {
		return ""
	}
	var name, pathVal string
	inProfileSection := false
	for _, line := range strings.Split(string(raw), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			inProfileSection = strings.Contains(trimmed, "Profile")
			name, pathVal = "", ""
			continue
		}
		if !inProfileSection {
			continue
		}
		k, v, ok := strings.Cut(trimmed, "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "Name":
			name = strings.TrimSpace(v)
		case "Path":
			pathVal = strings.TrimSpace(v)
		case "Default":
			if strings.TrimSpace(v) == "1" && name != "" {
				return name
			}
		}
	}
	if pathVal != "" {
		return filepath.Base(pathVal) // Path 可能含相对子目录
	}
	return ""
}

// FirefoxPlaces 打开（只读复制）places.sqlite 的查询句柄。
// Firefox 运行时锁定原文件，因此先复制到临时目录再打开。
type FirefoxPlaces struct {
	db    *sql.DB
	tmpDB string
}

// OpenPlacesCopy 复制 profile 目录下的 places.sqlite 到临时文件并只读打开。
func OpenPlacesCopy(profileDir string) (*FirefoxPlaces, error) {
	src := filepath.Join(profileDir, "places.sqlite")
	if _, err := os.Stat(src); err != nil {
		return nil, fmt.Errorf("bookmarks: places.sqlite 不存在: %w", err)
	}
	tmp, err := os.CreateTemp("", "places-*.sqlite")
	if err != nil {
		return nil, fmt.Errorf("bookmarks: 创建临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()

	if err := copyFile(tmpPath, src); err != nil {
		os.Remove(tmpPath)
		return nil, err
	}
	// WAL 模式下未合并数据在 -wal 文件，一并复制提高完整度
	if _, err := os.Stat(src + "-wal"); err == nil {
		_ = copyFile(tmpPath+"-wal", src+"-wal")
	}

	db, err := sql.Open("sqlite", "file:"+tmpPath+"?mode=ro")
	if err != nil {
		os.Remove(tmpPath)
		return nil, fmt.Errorf("bookmarks: 打开 places 副本失败: %w", err)
	}
	return &FirefoxPlaces{db: db, tmpDB: tmpPath}, nil
}

// Close 关闭连接并清理临时文件。
func (f *FirefoxPlaces) Close() error {
	if f.db != nil {
		f.db.Close()
	}
	if f.tmpDB != "" {
		os.Remove(f.tmpDB)
		os.Remove(f.tmpDB + "-wal")
		os.Remove(f.tmpDB + "-shm")
	}
	return nil
}

// BookmarkRoots 查询全部书签，按"书签工具栏 / 书签菜单 / 其他书签"归组返回。
// moz_bookmarks.type: 1=书签 URL, 2=文件夹；根容器 parent<=1。
func (f *FirefoxPlaces) BookmarkRoots() ([]*Node, error) {
	rows, err := f.db.Query(`
		SELECT b.id, b.parent, b.type, b.title, p.url
		FROM moz_bookmarks b
		LEFT JOIN moz_places p ON p.id = b.fk
		WHERE b.type IN (1, 2)`)
	if err != nil {
		return nil, fmt.Errorf("bookmarks: 查询书签失败: %w", err)
	}
	type rawRow struct {
		id     int64
		parent int64
		typ    int64
		title  string
		url    sql.NullString
	}
	var all []rawRow
	for rows.Next() {
		var r rawRow
		if err := rows.Scan(&r.id, &r.parent, &r.typ, &r.title, &r.url); err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	nodes := make(map[int64]*Node, len(all))
	for _, r := range all {
		n := &Node{Name: r.title}
		if r.typ == 1 && r.url.Valid {
			n.URL = r.url.String
		}
		nodes[r.id] = n
	}

	// 根容器 ID（toolbar/menu/unfiled 三个文件夹的 id）
	toolbarID, menuID, unfiledID := int64(-1), int64(-1), int64(-1)
	for _, r := range all {
		if r.typ == 2 && r.parent <= 1 {
			switch r.title {
			case "toolbar":
				toolbarID = r.id
			case "menu":
				menuID = r.id
			case "unfiled":
				unfiledID = r.id
			}
		}
	}

	underToolbar := &Node{Name: "书签工具栏"}
	underMenu := &Node{Name: "书签菜单"}
	underOther := &Node{Name: "其他书签"}
	for _, r := range all {
		n := nodes[r.id]
		if n == nil || r.parent <= 1 { // 跳过根容器自身
			continue
		}
		switch r.parent {
		case toolbarID:
			underToolbar.Children = append(underToolbar.Children, n)
		case menuID:
			underMenu.Children = append(underMenu.Children, n)
		case unfiledID:
			underOther.Children = append(underOther.Children, n)
		default:
			if p, ok := nodes[r.parent]; ok {
				p.Children = append(p.Children, n)
			}
		}
	}

	var roots []*Node
	for _, grp := range []*Node{underToolbar, underMenu, underOther} {
		if len(grp.Children) > 0 {
			roots = append(roots, grp)
		}
	}
	return roots, nil
}

// FirefoxBookmarks 解析 Firefox 书签（appData = %APPDATA%）。
func FirefoxBookmarks(appData string) ([]*Node, error) {
	return firefoxBookmarksCommon(appData, "Mozilla/Firefox")
}

// ZenBookmarks 解析 Zen Browser 书签（Zen 是 Firefox 分支，Profile 在 zen-browser-data）。
func ZenBookmarks(appData string) ([]*Node, error) {
	return firefoxBookmarksCommon(appData, "zen-browser-data")
}

func firefoxBookmarksCommon(appData, rootsDir string) ([]*Node, error) {
	profileDir, err := firefoxProfileDir(appData, rootsDir)
	if err != nil {
		return nil, err
	}
	places, err := OpenPlacesCopy(profileDir)
	if err != nil {
		return nil, err
	}
	defer places.Close()
	return places.BookmarkRoots()
}

// ProcessNames 浏览器进程名（Windows，小写匹配）。
var ProcessNames = map[BrowserKind][]string{
	BrowserEdge:    {"msedge.exe"},
	BrowserChrome:  {"chrome.exe"},
	BrowserZen:     {"zen.exe"},
	BrowserFirefox: {"firefox.exe"},
}

func copyFile(dst, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("bookmarks: 读取 %s 失败: %w", src, err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("bookmarks: 写入 %s 失败: %w", dst, err)
	}
	return nil
}
