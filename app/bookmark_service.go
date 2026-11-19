package app

import (
	"fmt"
	"sort"
	"strings"

	"my-web-mark/internal/bookmarks"
	"my-web-mark/internal/store"
)

// BookmarkService 书签服务。
type BookmarkService struct {
	sh *shared
}

// BookmarkInput 创建/更新书签入参。
type BookmarkInput struct {
	ThemeID int64    `json:"themeId"`
	Title   string   `json:"title"`
	URL     string   `json:"url"`
	Favicon string   `json:"favicon"`
	Note    string   `json:"note"`
	Pinned  bool     `json:"pinned"`
	TagIDs  []int64  `json:"tagIds"` // 关联标签（整体替换）
}

func (in *BookmarkInput) validate() error {
	if in.ThemeID <= 0 {
		return fmt.Errorf("app: 必须选择主题")
	}
	if in.URL == "" {
		return fmt.Errorf("app: URL 不能为空")
	}
	if !strings.Contains(in.URL, "://") {
		in.URL = "https://" + in.URL // 容错：自动补协议
	}
	if in.Title == "" {
		in.Title = in.URL
	}
	return nil
}

// ListByTheme 返回某主题下的书签。
func (b *BookmarkService) ListByTheme(themeID int64) ([]store.Bookmark, error) {
	return b.sh.store.ListBookmarksByTheme(themeID)
}

// ListAll 返回全部书签（2.5D 场景一次性加载）。
func (b *BookmarkService) ListAll() ([]store.Bookmark, error) {
	return b.sh.store.ListAllBookmarks()
}

// Create 新增书签。
func (b *BookmarkService) Create(in BookmarkInput) (*store.Bookmark, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	bm := &store.Bookmark{
		ThemeID: in.ThemeID, Title: in.Title, URL: in.URL,
		Favicon: in.Favicon, Note: in.Note, Pinned: in.Pinned,
		Source: "manual",
	}
	if err := b.sh.store.CreateBookmark(bm); err != nil {
		return nil, err
	}
	if len(in.TagIDs) > 0 {
		if err := b.sh.store.SetBookmarkTags(bm.ID, in.TagIDs); err != nil {
			return nil, err
		}
	}
	return b.sh.store.GetBookmark(bm.ID)
}

// Update 更新书签。
func (b *BookmarkService) Update(id int64, in BookmarkInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	if err := b.sh.store.UpdateBookmark(&store.Bookmark{
		ID: id, ThemeID: in.ThemeID, Title: in.Title, URL: in.URL,
		Favicon: in.Favicon, Note: in.Note, Pinned: in.Pinned,
	}); err != nil {
		return err
	}
	return b.sh.store.SetBookmarkTags(id, in.TagIDs)
}

// Delete 删除书签。
func (b *BookmarkService) Delete(id int64) error {
	return b.sh.store.DeleteBookmark(id)
}

// BookmarkSortInput 拖拽排序入参：按数组顺序写入 sort。
type BookmarkSortInput struct {
	IDs []int64 `json:"ids"`
}

// Sort 批量保存书签顺序（同主题内，置顶组在前由前端保证）。
func (b *BookmarkService) Sort(in BookmarkSortInput) error {
	if len(in.IDs) == 0 {
		return nil
	}
	return b.sh.store.SortBookmarks(in.IDs)
}

// --- 浏览器导入 ---

// BrowserBookmark 浏览器书签扁平条目（导入预览用）。
type BrowserBookmark struct {
	Title  string `json:"title"`
	URL    string `json:"url"`
	Folder string `json:"folder"`
}

// BrowserStatus 浏览器检测状态（包装 internal/bookmarks.Status）。
type BrowserStatus = bookmarks.Status

// DetectBrowsers 检测各浏览器运行状态与可导入书签数。
func (b *BookmarkService) DetectBrowsers() []BrowserStatus {
	dirs := bookmarks.DefaultAppDirs()
	var out []BrowserStatus
	for _, kind := range bookmarks.AllBrowsers {
		st := BrowserStatus{
			Browser:   kind,
			Name:      bookmarks.DisplayNames[kind],
			Bookmarks: -1,
		}
		var roots []*bookmarks.Node
		var err error
		switch kind {
		case bookmarks.BrowserEdge:
			roots, err = b.resolveEdge(dirs)
		case bookmarks.BrowserChrome:
			roots, err = b.resolveChrome(dirs)
		case bookmarks.BrowserZen:
			roots, err = b.resolveZen(dirs)
		case bookmarks.BrowserFirefox:
			roots, err = b.resolveFirefox(dirs)
		}
		if err != nil {
			if err == bookmarks.ErrNoProfile {
				st.Error = "未安装或未找到数据目录"
			} else {
				st.Error = err.Error()
			}
		} else {
			st.Installed = true
			st.Bookmarks = len(bookmarks.Flatten(roots...))
		}
		if running, err := bookmarks.DetectRunning(); err == nil {
			for _, r := range running {
				if r == kind {
					st.Running = true
				}
			}
		}
		out = append(out, st)
	}
	return out
}

// resolve 系列：解析各浏览器书签根节点。
func (b *BookmarkService) resolveEdge(d bookmarks.AppDirs) ([]*bookmarks.Node, error) {
	return bookmarks.EdgeBookmarks(d.LocalAppData)
}
func (b *BookmarkService) resolveChrome(d bookmarks.AppDirs) ([]*bookmarks.Node, error) {
	return bookmarks.ChromeBookmarks(d.LocalAppData)
}
func (b *BookmarkService) resolveZen(d bookmarks.AppDirs) ([]*bookmarks.Node, error) {
	return bookmarks.ZenBookmarks(d.AppData)
}
func (b *BookmarkService) resolveFirefox(d bookmarks.AppDirs) ([]*bookmarks.Node, error) {
	return bookmarks.FirefoxBookmarks(d.AppData)
}

// ImportPreview 导入预览结果：按顶层文件夹归组的扁平书签。
type ImportPreview struct {
	Browser   string           `json:"browser"`
	Items     []BrowserBookmark `json:"items"`
	Folders   []string         `json:"folders"` // 顶层文件夹名（建议映射为主题）
}

// PreviewImport 预览某浏览器的书签（不落库）。
func (b *BookmarkService) PreviewImport(browser string) (*ImportPreview, error) {
	dirs := bookmarks.DefaultAppDirs()
	var roots []*bookmarks.Node
	var err error
	switch bookmarks.BrowserKind(browser) {
	case bookmarks.BrowserEdge:
		roots, err = b.resolveEdge(dirs)
	case bookmarks.BrowserChrome:
		roots, err = b.resolveChrome(dirs)
	case bookmarks.BrowserZen:
		roots, err = b.resolveZen(dirs)
	case bookmarks.BrowserFirefox:
		roots, err = b.resolveFirefox(dirs)
	default:
		return nil, fmt.Errorf("app: 不支持的浏览器 %q", browser)
	}
	if err != nil {
		return nil, err
	}
	flat := bookmarks.Flatten(roots...)
	preview := &ImportPreview{
		Browser: browser,
		Items:   make([]BrowserBookmark, 0, len(flat)),
	}
	folderSet := map[string]bool{}
	for _, f := range flat {
		preview.Items = append(preview.Items, BrowserBookmark{
			Title: f.Title, URL: f.URL, Folder: f.Folder,
		})
		if top := firstSegment(f.Folder); top != "" {
			folderSet[top] = true
		}
	}
	for folder := range folderSet {
		preview.Folders = append(preview.Folders, folder)
	}
	sort.Strings(preview.Folders)
	return preview, nil
}

// ImportPlanItem 导入计划单项：浏览器文件夹 -> 目标主题。
type ImportPlanItem struct {
	Folder     string `json:"folder"`     // 浏览器侧顶层文件夹；"" 表示未分组
	TargetThemeID int64  `json:"targetThemeId"` // 导入到哪个主题（0 = 自动建同名主题）
}

// ImportPlan 导入计划。
type ImportPlan struct {
	Browser string           `json:"browser"`
	Items   []ImportPlanItem `json:"items"`
}

// ImportResult 导入执行结果。
type ImportResult struct {
	Imported int      `json:"imported"` // 成功导入条数
	Skipped  int      `json:"skipped"`  // 重复跳过条数
	Errors   []string `json:"errors,omitempty"`
}

// ExecuteImport 按计划导入书签。folderToTheme 为空时自动为每个顶层文件夹
// 创建同名主题（无文件夹的书签进入"导入"兜底主题）。
func (b *BookmarkService) ExecuteImport(plan ImportPlan) (*ImportResult, error) {
	preview, err := b.PreviewImport(plan.Browser)
	if err != nil {
		return nil, err
	}
	// 计划映射：folder -> themeID
	planMap := make(map[string]int64, len(plan.Items))
	for _, it := range plan.Items {
		planMap[it.Folder] = it.TargetThemeID
	}

	// 主题不存在时按名称自动创建
	ensureTheme := func(name string) (int64, error) {
		themes, err := t_list(b.sh.store)
		if err != nil {
			return 0, err
		}
		for _, th := range themes {
			if th.Name == name {
				return th.ID, nil
			}
		}
		th := &store.Theme{Name: name}
		if err := b.sh.store.CreateTheme(th); err != nil {
			return 0, err
		}
		return th.ID, nil
	}

	res := &ImportResult{}
	for _, item := range preview.Items {
		folder := firstSegment(item.Folder)
		if folder == "" {
			folder = "导入"
		}
		themeID, ok := planMap[folder]
		if !ok { // 计划未覆盖 -> 自动建同名主题
			var err error
			themeID, err = ensureTheme(folder)
			if err != nil {
				res.Errors = append(res.Errors, fmt.Sprintf("创建主题 %s 失败: %v", folder, err))
				continue
			}
		}
		if themeID == 0 { // 计划里显式为 0 -> 跳过
			res.Skipped++
			continue
		}
		bm := &store.Bookmark{
			ThemeID: themeID, Title: item.Title, URL: item.URL,
			Source: plan.Browser,
		}
		if err := b.sh.store.CreateBookmark(bm); err != nil {
			if strings.Contains(err.Error(), "已存在") {
				res.Skipped++
			} else {
				res.Errors = append(res.Errors, fmt.Sprintf("导入 %s 失败: %v", item.URL, err))
			}
			continue
		}
		res.Imported++
	}
	return res, nil
}

// firstSegment 取路径首段。
func firstSegment(path string) string {
	if path == "" {
		return ""
	}
	if i := strings.IndexByte(path, '/'); i >= 0 {
		return path[:i]
	}
	return path
}

// t_list 主题列表包装（避免循环引用命名）。
func t_list(st *store.Store) ([]store.Theme, error) {
	return st.ListThemes()
}

