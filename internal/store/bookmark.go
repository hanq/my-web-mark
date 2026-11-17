package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// CreateBookmark 新增书签；theme+url 唯一约束防重复。
func (s *Store) CreateBookmark(b *Bookmark) error {
	res, err := s.db.Exec(
		`INSERT INTO bookmark (theme_id, title, url, favicon, note, pinned, source)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		b.ThemeID, b.Title, b.URL, b.Favicon, b.Note, boolToInt(b.Pinned), b.Source)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("store: 书签已存在: %w", err)
		}
		return fmt.Errorf("store: 新增书签失败: %w", err)
	}
	b.ID, err = res.LastInsertId()
	return err
}

// UpdateBookmark 更新书签字段。
func (s *Store) UpdateBookmark(b *Bookmark) error {
	res, err := s.db.Exec(
		`UPDATE bookmark SET theme_id = ?, title = ?, url = ?, favicon = ?, note = ?, pinned = ?
		 WHERE id = ?`,
		b.ThemeID, b.Title, b.URL, b.Favicon, b.Note, boolToInt(b.Pinned), b.ID)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			return fmt.Errorf("store: 目标主题下已有相同 URL: %w", err)
		}
		return fmt.Errorf("store: 更新书签失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteBookmark 按 id 删除书签。
func (s *Store) DeleteBookmark(id int64) error {
	res, err := s.db.Exec(`DELETE FROM bookmark WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: 删除书签失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetBookmark 按 id 查询单条书签。
func (s *Store) GetBookmark(id int64) (*Bookmark, error) {
	row := s.db.QueryRow(
		`SELECT id, theme_id, title, url, favicon, note, pinned, source, created_at
		 FROM bookmark WHERE id = ?`, id)
	return scanBookmark(row.Scan)
}

// ListBookmarksByTheme 查询某主题下全部书签，置顶优先，其余按创建时间倒序。
func (s *Store) ListBookmarksByTheme(themeID int64) ([]Bookmark, error) {
	rows, err := s.db.Query(
		`SELECT id, theme_id, title, url, favicon, note, pinned, source, created_at
		 FROM bookmark WHERE theme_id = ?
		 ORDER BY pinned DESC, created_at DESC`, themeID)
	if err != nil {
		return nil, fmt.Errorf("store: 查询书签失败: %w", err)
	}
	return collectBookmarks(rows)
}

// ListAllBookmarks 返回全部书签（2.5D 场景一次性加载用）。
func (s *Store) ListAllBookmarks() ([]Bookmark, error) {
	rows, err := s.db.Query(
		`SELECT id, theme_id, title, url, favicon, note, pinned, source, created_at
		 FROM bookmark ORDER BY theme_id, pinned DESC, created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("store: 查询书签失败: %w", err)
	}
	return collectBookmarks(rows)
}

// CountBookmarksByTheme 返回各主题书签数量（theme_id -> count）。
func (s *Store) CountBookmarksByTheme() (map[int64]int, error) {
	rows, err := s.db.Query(`SELECT theme_id, COUNT(*) FROM bookmark GROUP BY theme_id`)
	if err != nil {
		return nil, fmt.Errorf("store: 统计书签失败: %w", err)
	}
	defer rows.Close()
	out := make(map[int64]int)
	for rows.Next() {
		var tid int64
		var n int
		if err := rows.Scan(&tid, &n); err != nil {
			return nil, err
		}
		out[tid] = n
	}
	return out, rows.Err()
}

type rowScanner func(dest ...any) error

func scanBookmark(scan rowScanner) (*Bookmark, error) {
	var b Bookmark
	var pinned int
	var created sql.NullString
	err := scan(&b.ID, &b.ThemeID, &b.Title, &b.URL, &b.Favicon, &b.Note, &pinned, &b.Source, &created)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	b.Pinned = pinned == 1
	b.CreatedAt = parseTime(created.String)
	return &b, nil
}

func collectBookmarks(rows *sql.Rows) ([]Bookmark, error) {
	defer rows.Close()
	var out []Bookmark
	for rows.Next() {
		b, err := scanBookmark(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
