package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"my-web-mark/internal/bookmarks"
	"my-web-mark/internal/store"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExportService 主题书签数据导出/导入。
// 导出格式：html（Netscape 标准，浏览器可导入）/ json（完整备份，含标签备注置顶）。
// 导入自动检测格式，落为新主题（同名自动加后缀）。
type ExportService struct {
	sh *shared
}

// ImportThemeResult 导入结果统计。
type ImportThemeResult struct {
	ThemeName string `json:"themeName"`
	ThemeID   int64  `json:"themeId"`
	Imported  int    `json:"imported"`
	Skipped   int    `json:"skipped"`
	TagCount  int    `json:"tagCount"`
}

// ExportTheme 导出指定主题。返回写入的文件路径；用户取消对话框返回空串。
func (e *ExportService) ExportTheme(themeID int64, format string) (string, error) {
	if format != "html" && format != "json" {
		return "", fmt.Errorf("app: 不支持的导出格式 %q", format)
	}
	th, err := e.sh.store.GetTheme(themeID)
	if err != nil {
		return "", fmt.Errorf("app: 主题不存在: %w", err)
	}
	bms, err := e.sh.store.ListBookmarksByTheme(themeID)
	if err != nil {
		return "", err
	}
	items := make([]bookmarks.ExportBookmark, 0, len(bms))
	for _, b := range bms {
		tags := make([]string, 0, len(b.Tags))
		for _, tg := range b.Tags {
			tags = append(tags, tg.Name)
		}
		folder := ""
		if len(tags) > 0 {
			folder = tags[0]
		}
		items = append(items, bookmarks.ExportBookmark{
			Title: b.Title, URL: b.URL, Note: b.Note,
			Pinned: b.Pinned, Folder: folder, Tags: tags,
		})
	}

	var data []byte
	var fname string
	if format == "json" {
		data, err = bookmarks.MarshalThemeFile(bookmarks.ExportThemeMeta{Name: th.Name, Color: th.Color}, items)
		fname = sanitizeFilename(th.Name) + "-bookmarks.json"
	} else {
		data = []byte(bookmarks.RenderNetscapeHTML(th.Name, items))
		fname = sanitizeFilename(th.Name) + "-bookmarks.html"
	}
	if err != nil {
		return "", err
	}

	display := "HTML 书签文件"
	if format == "json" {
		display = "JSON 备份"
	}
	path, err := wailsruntime.SaveFileDialog(e.sh.ctx, wailsruntime.SaveDialogOptions{
		Title:           "导出主题书签",
		DefaultFilename: fname,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: display, Pattern: "*." + format},
		},
	})
	if err != nil {
		return "", fmt.Errorf("app: 保存对话框失败: %w", err)
	}
	if path == "" {
		return "", nil // 用户取消
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("app: 写文件失败: %w", err)
	}
	return path, nil
}

// ImportTheme 打开文件对话框并导入主题书签。用户取消返回 (nil, nil)。
func (e *ExportService) ImportTheme() (*ImportThemeResult, error) {
	path, err := wailsruntime.OpenFileDialog(e.sh.ctx, wailsruntime.OpenDialogOptions{
		Title: "导入主题书签文件",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "书签文件 (HTML/JSON)", Pattern: "*.html;*.htm;*.json"},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("app: 打开对话框失败: %w", err)
	}
	if path == "" {
		return nil, nil // 用户取消
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("app: 读取文件失败: %w", err)
	}
	content := string(raw)
	fallbackName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))

	var (
		themeName, themeColor string
		items                 []bookmarks.ExportBookmark
		pinnedSupport         bool
	)
	if bookmarks.DetectExportFormat(content) == "json" {
		f, perr := bookmarks.ParseThemeFile(raw)
		if perr != nil {
			return nil, perr
		}
		themeName, themeColor, items, pinnedSupport = f.Theme.Name, f.Theme.Color, f.Bookmarks, true
	} else {
		name, parsed, perr := bookmarks.ParseNetscapeHTML(content)
		if perr != nil {
			return nil, perr
		}
		themeName, items = name, parsed
		for i := range items {
			if items[i].Folder != "" {
				items[i].Tags = []string{items[i].Folder}
			}
		}
	}
	if themeName == "" {
		themeName = fallbackName
	}
	return e.applyImport(themeName, themeColor, items, pinnedSupport)
}

// applyImport 落库：建主题（冲突加后缀）-> 建标签 -> 建书签。可独立测试。
func (e *ExportService) applyImport(themeName, themeColor string, items []bookmarks.ExportBookmark, usePinned bool) (*ImportThemeResult, error) {
	res := &ImportThemeResult{ThemeName: themeName}

	// 主题名冲突 → 追加后缀（保持可重复导入不覆盖）
	name := themeName
	for {
		existing, err := e.sh.store.ListThemes()
		if err != nil {
			return nil, err
		}
		conflict := false
		for _, th := range existing {
			if th.Name == name {
				conflict = true
				break
			}
		}
		if !conflict {
			break
		}
		name += " (导入)"
	}
	res.ThemeName = name
	th := &store.Theme{Name: name}
	if themeColor != "" {
		th.Color = themeColor
	}
	if err := e.sh.store.CreateTheme(th); err != nil {
		return nil, err
	}
	res.ThemeID = th.ID

	// 标签：同名复用，缺则建；Tags 为空时回退 Folder（HTML 解析只填 Folder）
	tagIDs := map[string]int64{}
	for _, it := range items {
		tagNames := it.Tags
		if len(tagNames) == 0 && it.Folder != "" {
			tagNames = []string{it.Folder}
		}
		for _, tn := range tagNames {
			if tn == "" {
				continue
			}
			if _, ok := tagIDs[tn]; ok {
				continue
			}
			tg := &store.Tag{ThemeID: th.ID, Name: tn}
			if err := e.sh.store.CreateTag(tg); err == nil {
				tagIDs[tn] = tg.ID
			} else {
				// 同名已存在：查回 id
				tags, _ := e.sh.store.ListTagsByTheme(th.ID)
				for _, t := range tags {
					if t.Name == tn {
						tagIDs[tn] = t.ID
					}
				}
			}
		}
	}
	res.TagCount = len(tagIDs)

	for _, it := range items {
		u := strings.TrimSpace(it.URL)
		if u == "" {
			continue
		}
		if !strings.Contains(u, "://") {
			u = "https://" + u
		}
		title := strings.TrimSpace(it.Title)
		if title == "" {
			title = u
		}
		tagNames := it.Tags
		if len(tagNames) == 0 && it.Folder != "" {
			tagNames = []string{it.Folder}
		}
		ids := make([]int64, 0, len(tagNames))
		for _, tn := range tagNames {
			if id, ok := tagIDs[tn]; ok {
				ids = append(ids, id)
			}
		}
		bm := &store.Bookmark{
			ThemeID: th.ID, Title: title, URL: u,
			Note: it.Note, Source: "import",
		}
		if usePinned {
			bm.Pinned = it.Pinned
		}
		if err := e.sh.store.CreateBookmark(bm); err != nil {
			if strings.Contains(err.Error(), "已存在") {
				res.Skipped++
				continue
			}
			return nil, err
		}
		if len(ids) > 0 {
			if err := e.sh.store.SetBookmarkTags(bm.ID, ids); err != nil {
				return nil, err
			}
		}
		res.Imported++
	}
	return res, nil
}

// sanitizeFilename 替换 Windows 文件名非法字符。
func sanitizeFilename(name string) string {
	repl := strings.NewReplacer(
		"\\", "_", "/", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_",
	)
	out := strings.TrimSpace(repl.Replace(name))
	if out == "" {
		out = "theme"
	}
	return out
}
