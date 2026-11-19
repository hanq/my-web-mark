package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// Tag 主题内标签，用于书签二次分类。
type Tag struct {
	ID      int64  `json:"id"`
	ThemeID int64  `json:"themeId"`
	Name    string `json:"name"`
	Color   string `json:"color"`
	Sort    int    `json:"sort"`
}

// CreateTag 新建标签；同主题下名称唯一。
func (s *Store) CreateTag(t *Tag) error {
	res, err := s.db.Exec(
		`INSERT INTO tag (theme_id, name, color, sort) VALUES (?, ?, ?, ?)`,
		t.ThemeID, t.Name, t.Color, t.Sort)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("store: 标签已存在: %w", err)
		}
		return fmt.Errorf("store: 新建标签失败: %w", err)
	}
	t.ID, err = res.LastInsertId()
	return err
}

// UpdateTag 更新标签名称/颜色/排序。
func (s *Store) UpdateTag(t *Tag) error {
	res, err := s.db.Exec(
		`UPDATE tag SET name = ?, color = ?, sort = ? WHERE id = ?`,
		t.Name, t.Color, t.Sort, t.ID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("store: 标签已存在: %w", err)
		}
		return fmt.Errorf("store: 更新标签失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteTag 删除标签；书签关联随外键级联清除。
func (s *Store) DeleteTag(id int64) error {
	res, err := s.db.Exec(`DELETE FROM tag WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: 删除标签失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListTagsByTheme 返回某主题全部标签（按 sort、id 排序）。
func (s *Store) ListTagsByTheme(themeID int64) ([]Tag, error) {
	rows, err := s.db.Query(
		`SELECT id, theme_id, name, color, sort FROM tag WHERE theme_id = ? ORDER BY sort, id`,
		themeID)
	if err != nil {
		return nil, fmt.Errorf("store: 查询标签失败: %w", err)
	}
	return collectTags(rows)
}

// ListAllTags 返回全部主题的标签（2.5D 全局筛选条用）。
func (s *Store) ListAllTags() ([]Tag, error) {
	rows, err := s.db.Query(
		`SELECT id, theme_id, name, color, sort FROM tag ORDER BY theme_id, sort, id`)
	if err != nil {
		return nil, fmt.Errorf("store: 查询标签失败: %w", err)
	}
	return collectTags(rows)
}

// SetBookmarkTags 整体替换书签的标签关联。
func (s *Store) SetBookmarkTags(bookmarkID int64, tagIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: 开启事务失败: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM bookmark_tag WHERE bookmark_id = ?`, bookmarkID); err != nil {
		return err
	}
	for _, tid := range tagIDs {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO bookmark_tag (bookmark_id, tag_id) VALUES (?, ?)`,
			bookmarkID, tid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// attachTags 批量为书签填充 tags 字段（一次查询 + 内存分组）。
func (s *Store) attachTags(bookmarks []Bookmark) error {
	if len(bookmarks) == 0 {
		return nil
	}
	rows, err := s.db.Query(`
		SELECT bt.bookmark_id, t.id, t.theme_id, t.name, t.color, t.sort
		FROM bookmark_tag bt
		JOIN tag t ON t.id = bt.tag_id
		ORDER BY t.sort, t.id`)
	if err != nil {
		return fmt.Errorf("store: 查询书签标签关联失败: %w", err)
	}
	defer rows.Close()

	idx := make(map[int64]int, len(bookmarks))
	for i, b := range bookmarks {
		idx[b.ID] = i
		b.Tags = []Tag{} // 初始化为空数组，前端序列化稳定
		bookmarks[i].Tags = b.Tags
	}
	for rows.Next() {
		var bmID int64
		var t Tag
		if err := rows.Scan(&bmID, &t.ID, &t.ThemeID, &t.Name, &t.Color, &t.Sort); err != nil {
			return err
		}
		if i, ok := idx[bmID]; ok {
			bookmarks[i].Tags = append(bookmarks[i].Tags, t)
		}
	}
	return rows.Err()
}

func collectTags(rows *sql.Rows) ([]Tag, error) {
	defer rows.Close()
	var out []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.ThemeID, &t.Name, &t.Color, &t.Sort); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
