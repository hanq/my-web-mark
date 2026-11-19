package store

import (
	"database/sql"
	"fmt"
)

// accountColumns 账号查询列（顺序与 scanAccount 一致）。
const accountColumns = `id, theme_id, kind, bookmark_id, site, username, password_cipher, url, note, created_at`

// CreateAccount 新增账号；passwordCipher 必须为调用方加密后的密文（无密码时传空）。
func (s *Store) CreateAccount(a *Account) error {
	res, err := s.db.Exec(
		`INSERT INTO account (theme_id, kind, bookmark_id, site, username, password_cipher, url, note)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.ThemeID, a.Kind, nullableID(a.BookmarkID), a.Site, a.Username, blobOrEmpty(a.PasswordCipher), a.URL, a.Note)
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
			`UPDATE account SET theme_id = ?, kind = ?, bookmark_id = ?, site = ?, username = ?, url = ?, note = ?
			 WHERE id = ?`,
			a.ThemeID, a.Kind, nullableID(a.BookmarkID), a.Site, a.Username, a.URL, a.Note, a.ID)
	} else {
		res, err = s.db.Exec(
			`UPDATE account SET theme_id = ?, kind = ?, bookmark_id = ?, site = ?, username = ?,
			 password_cipher = ?, url = ?, note = ? WHERE id = ?`,
			a.ThemeID, a.Kind, nullableID(a.BookmarkID), a.Site, a.Username, a.PasswordCipher, a.URL, a.Note, a.ID)
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

// GetAccount 按 id 查询单条账号（含密文与标签）。
func (s *Store) GetAccount(id int64) (*Account, error) {
	row := s.db.QueryRow(`SELECT `+accountColumns+` FROM account WHERE id = ?`, id)
	a, err := scanAccount(row.Scan)
	if err != nil {
		return nil, err
	}
	list := []Account{*a}
	if err := s.attachAccountTags(list); err != nil {
		return nil, err
	}
	return &list[0], nil
}

// ListAccountsByTheme 查询某主题下全部账号（含密文与标签，由服务层决定是否解密）。
func (s *Store) ListAccountsByTheme(themeID int64) ([]Account, error) {
	rows, err := s.db.Query(
		`SELECT `+accountColumns+` FROM account WHERE theme_id = ? ORDER BY created_at DESC`, themeID)
	if err != nil {
		return nil, fmt.Errorf("store: 查询账号失败: %w", err)
	}
	accounts, err := collectAccounts(rows)
	if err != nil {
		return nil, err
	}
	if err := s.attachAccountTags(accounts); err != nil {
		return nil, err
	}
	return accounts, nil
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

// SetAccountTags 整体替换账号的标签关联。
func (s *Store) SetAccountTags(accountID int64, tagIDs []int64) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: 开启事务失败: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM account_tag WHERE account_id = ?`, accountID); err != nil {
		return err
	}
	for _, tid := range tagIDs {
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO account_tag (account_id, tag_id) VALUES (?, ?)`,
			accountID, tid); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// attachAccountTags 批量为账号填充 tags 字段（一次查询 + 内存分组）。
func (s *Store) attachAccountTags(accounts []Account) error {
	for i := range accounts {
		accounts[i].Tags = []Tag{} // 初始化为空数组，前端序列化稳定
	}
	if len(accounts) == 0 {
		return nil
	}
	rows, err := s.db.Query(`
		SELECT at.account_id, t.id, t.theme_id, t.name, t.color, t.sort
		FROM account_tag at
		JOIN tag t ON t.id = at.tag_id
		ORDER BY t.sort, t.id`)
	if err != nil {
		return fmt.Errorf("store: 查询账号标签关联失败: %w", err)
	}
	defer rows.Close()

	idx := make(map[int64]int, len(accounts))
	for i := range accounts {
		idx[accounts[i].ID] = i
	}
	for rows.Next() {
		var accID int64
		var t Tag
		if err := rows.Scan(&accID, &t.ID, &t.ThemeID, &t.Name, &t.Color, &t.Sort); err != nil {
			return err
		}
		if i, ok := idx[accID]; ok {
			accounts[i].Tags = append(accounts[i].Tags, t)
		}
	}
	return rows.Err()
}

// nullableID 把可空的外键 id 转成 SQL 参数。
func nullableID(id *int64) any {
	if id == nil || *id <= 0 {
		return nil
	}
	return *id
}

// blobOrEmpty 保证 BLOB NOT NULL 列写入空切片而不是 NULL。
func blobOrEmpty(b []byte) []byte {
	if b == nil {
		return []byte{}
	}
	return b
}

func scanAccount(scan rowScanner) (*Account, error) {
	var a Account
	var bookmarkID sql.NullInt64
	var created sql.NullString
	err := scan(&a.ID, &a.ThemeID, &a.Kind, &bookmarkID, &a.Site, &a.Username,
		&a.PasswordCipher, &a.URL, &a.Note, &created)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if bookmarkID.Valid && bookmarkID.Int64 > 0 {
		id := bookmarkID.Int64
		a.BookmarkID = &id
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
