package store

import (
	"database/sql"
	"fmt"
)

// CreateAccount 新增账号；passwordCipher 必须为调用方加密后的密文。
func (s *Store) CreateAccount(a *Account) error {
	res, err := s.db.Exec(
		`INSERT INTO account (theme_id, site, username, password_cipher, url, note)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		a.ThemeID, a.Site, a.Username, a.PasswordCipher, a.URL, a.Note)
	if err != nil {
		return fmt.Errorf("store: 新增账号失败: %w", err)
	}
	a.ID, err = res.LastInsertId()
	return err
}

// UpdateAccount 更新账号字段；passwordCipher 为 nil 时保持原密文不变。
func (s *Store) UpdateAccount(a *Account) error {
	var (
		res sql.Result
		err error
	)
	if a.PasswordCipher == nil {
		res, err = s.db.Exec(
			`UPDATE account SET theme_id = ?, site = ?, username = ?, url = ?, note = ? WHERE id = ?`,
			a.ThemeID, a.Site, a.Username, a.URL, a.Note, a.ID)
	} else {
		res, err = s.db.Exec(
			`UPDATE account SET theme_id = ?, site = ?, username = ?, password_cipher = ?, url = ?, note = ? WHERE id = ?`,
			a.ThemeID, a.Site, a.Username, a.PasswordCipher, a.URL, a.Note, a.ID)
	}
	if err != nil {
		return fmt.Errorf("store: 更新账号失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteAccount 按 id 删除账号。
func (s *Store) DeleteAccount(id int64) error {
	res, err := s.db.Exec(`DELETE FROM account WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("store: 删除账号失败: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// GetAccount 按 id 查询单条账号（含密文）。
func (s *Store) GetAccount(id int64) (*Account, error) {
	row := s.db.QueryRow(
		`SELECT id, theme_id, site, username, password_cipher, url, note, created_at
		 FROM account WHERE id = ?`, id)
	return scanAccount(row.Scan)
}

// ListAccountsByTheme 查询某主题下全部账号（含密文，由服务层决定是否解密）。
func (s *Store) ListAccountsByTheme(themeID int64) ([]Account, error) {
	rows, err := s.db.Query(
		`SELECT id, theme_id, site, username, password_cipher, url, note, created_at
		 FROM account WHERE theme_id = ? ORDER BY created_at DESC`, themeID)
	if err != nil {
		return nil, fmt.Errorf("store: 查询账号失败: %w", err)
	}
	return collectAccounts(rows)
}

// CountAccountsByTheme 返回各主题账号数量。
func (s *Store) CountAccountsByTheme() (map[int64]int, error) {
	rows, err := s.db.Query(`SELECT theme_id, COUNT(*) FROM account GROUP BY theme_id`)
	if err != nil {
		return nil, fmt.Errorf("store: 统计账号失败: %w", err)
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

func scanAccount(scan rowScanner) (*Account, error) {
	var a Account
	var created sql.NullString
	err := scan(&a.ID, &a.ThemeID, &a.Site, &a.Username, &a.PasswordCipher, &a.URL, &a.Note, &created)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	a.CreatedAt = parseTime(created.String)
	return &a, nil
}

func collectAccounts(rows *sql.Rows) ([]Account, error) {
	defer rows.Close()
	var out []Account
	for rows.Next() {
		a, err := scanAccount(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}
