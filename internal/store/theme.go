package store

import (
	"database/sql"
	"fmt"
	"time"
)

// CreateTheme 新建主题；名称重复会返回唯一约束错误。
func (s *Store) CreateTheme(t *Theme) error {
	res, err := s.db.Exec(
		`INSERT INTO theme (name, color, icon, sort) VALUES (?, ?, ?, ?)`,
		t.Name, t.Color, t.Icon, t.Sort)
	if err != nil {
		return fmt.Errorf("store: 新建主题失败: %w", err)
	}
	t.ID, err = res.LastInsertId()
	return err
}

// UpdateTheme 更新主题的可编辑字段。
func (s *Store) UpdateTheme(t *Theme) error {
	res, err := s.db.Exec(
		`UPDATE theme SET name = ?, color = ?, icon = ?, sort = ? WHERE id = ?`,
		t.Name, t.Color, t.Icon, t.Sort, t.ID)
	if err != nil {
		return fmt.Errorf("store: 更新主题失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateThemeSort 仅更新排序字段。
func (s *Store) UpdateThemeSort(id int64, sort int) error {
	res, err := s.db.Exec(`UPDATE theme SET sort = ? WHERE id = ?`, sort, id)
	if err != nil {
		return fmt.Errorf("store: 更新主题排序失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteTheme 删除主题（级联删除其下书签与账号）。
func (s *Store) DeleteTheme(id int64) error {
	res, err := s.db.Exec(`DELETE FROM theme WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: 删除主题失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListThemes 返回全部主题，按 sort、id 排序。
func (s *Store) ListThemes() ([]Theme, error) {
	rows, err := s.db.Query(`SELECT id, name, color, icon, sort, created_at FROM theme ORDER BY sort, id`)
	if err != nil {
		return nil, fmt.Errorf("store: 查询主题失败: %w", err)
	}
	defer rows.Close()

	var out []Theme
	for rows.Next() {
		var t Theme
		var created sql.NullString
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.Icon, &t.Sort, &created); err != nil {
			return nil, err
		}
		t.CreatedAt = parseTime(created.String)
		out = append(out, t)
	}
	return out, rows.Err()
}

// parseTime 兼容 SQLite CURRENT_TIMESTAMP 与 datetime() 两种格式。
func parseTime(v string) time.Time {
	if v == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.DateTime, "2006-01-02 15:04:05.999999999-07:00", time.RFC3339} {
		if t, err := time.Parse(layout, v); err == nil {
			return t
		}
	}
	return time.Time{}
}
