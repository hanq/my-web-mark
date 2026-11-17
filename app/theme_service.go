package app

import (
	"fmt"
	"sort"

	"my-web-mark/internal/store"
)

// ThemeService 主题分组服务。
type ThemeService struct {
	sh *shared
}

// ThemeInput 创建/更新主题入参。
type ThemeInput struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Icon  string `json:"icon"`
}

func (in *ThemeInput) validate() error {
	if in.Name == "" {
		return fmt.Errorf("app: 主题名称不能为空")
	}
	return nil
}

// List 返回全部主题，附带各主题的书签与账号数量。
func (t *ThemeService) List() ([]ThemeSummary, error) {
	themes, err := t.sh.store.ListThemes()
	if err != nil {
		return nil, err
	}
	bmCounts, _ := t.sh.store.CountBookmarksByTheme()
	accCounts, _ := t.sh.store.CountAccountsByTheme()

	out := make([]ThemeSummary, 0, len(themes))
	for _, th := range themes {
		out = append(out, ThemeSummary{
			Theme:        th,
			BookmarkCount: bmCounts[th.ID],
			AccountCount:  accCounts[th.ID],
		})
	}
	return out, nil
}

// ThemeSummary 主题 + 统计信息。
type ThemeSummary struct {
	store.Theme
	BookmarkCount int `json:"bookmarkCount"`
	AccountCount  int `json:"accountCount"`
}

// Create 新建主题。
func (t *ThemeService) Create(in ThemeInput) (*store.Theme, error) {
	if err := in.validate(); err != nil {
		return nil, err
	}
	th := &store.Theme{Name: in.Name, Color: in.Color, Icon: in.Icon}
	// 新主题排到末尾
	if themes, err := t.sh.store.ListThemes(); err == nil {
		th.Sort = len(themes)
	}
	if err := t.sh.store.CreateTheme(th); err != nil {
		return nil, err
	}
	return th, nil
}

// Update 更新主题。
func (t *ThemeService) Update(id int64, in ThemeInput) error {
	if err := in.validate(); err != nil {
		return err
	}
	return t.sh.store.UpdateTheme(&store.Theme{
		ID: id, Name: in.Name, Color: in.Color, Icon: in.Icon, Sort: 0,
	})
}

// SortInput 主题排序入参：按传入顺序的 id 列表重排。
type SortInput struct {
	IDs []int64 `json:"ids"`
}

// Sort 按给定顺序重排主题。
func (t *ThemeService) Sort(in SortInput) error {
	for i, id := range in.IDs {
		if err := t.sh.store.UpdateThemeSort(id, i); err != nil {
			return err
		}
	}
	return nil
}

// Delete 删除主题及其全部书签/账号（级联）。
func (t *ThemeService) Delete(id int64) error {
	return t.sh.store.DeleteTheme(id)
}

// reorderThemes 辅助：确保 sort 字段一致性（预留）。
func reorderThemes(themes []store.Theme) []store.Theme {
	sort.Slice(themes, func(i, j int) bool { return themes[i].Sort < themes[j].Sort })
	return themes
}
