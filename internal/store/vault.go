package store

import (
	"database/sql"
	"fmt"
)

// GetVaultMeta 读取管理员凭据元数据；未设置时返回 ErrNotFound。
func (s *Store) GetVaultMeta() (*VaultMeta, error) {
	row := s.db.QueryRow(
		`SELECT salt, password_hash, kdf_params, created_at FROM vault_meta WHERE id = 1`)
	var m VaultMeta
	var created sql.NullString
	err := row.Scan(&m.Salt, &m.PasswordHash, &m.KDFParams, &created)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("store: 读取金库元数据失败: %w", err)
	}
	m.CreatedAt = parseTime(created.String)
	return &m, nil
}

// SaveVaultMeta 保存管理员凭据；若已存在则覆盖（用于修改管理员密码）。
func (s *Store) SaveVaultMeta(m *VaultMeta) error {
	_, err := s.db.Exec(
		`INSERT INTO vault_meta (id, salt, password_hash, kdf_params)
		 VALUES (1, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET salt = excluded.salt,
			password_hash = excluded.password_hash,
			kdf_params = excluded.kdf_params`,
		m.Salt, m.PasswordHash, m.KDFParams)
	if err != nil {
		return fmt.Errorf("store: 保存金库元数据失败: %w", err)
	}
	return nil
}

// HasVaultMeta 判断是否已设置管理员密码。
func (s *Store) HasVaultMeta() (bool, error) {
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM vault_meta WHERE id = 1`).Scan(&n)
	return n > 0, err
}

// RekeyAccounts 批量替换全部账号密码密文（修改管理员密码后用新密钥重加密）。
// 在一个事务中完成，失败即回滚。
func (s *Store) RekeyAccounts(reencrypt func(themeID int64, cipher []byte) ([]byte, error)) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: 开启事务失败: %w", err)
	}
	defer tx.Rollback()

	rows, err := tx.Query(`SELECT id, theme_id, password_cipher FROM account`)
	if err != nil {
		return err
	}
	type row struct {
		id      int64
		themeID int64
		cipher  []byte
	}
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.themeID, &r.cipher); err != nil {
			rows.Close()
			return err
		}
		all = append(all, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, r := range all {
		newCipher, err := reencrypt(r.themeID, r.cipher)
		if err != nil {
			return fmt.Errorf("store: 重加密账号 #%d 失败: %w", r.id, err)
		}
		if _, err := tx.Exec(`UPDATE account SET password_cipher = ? WHERE id = ?`, newCipher, r.id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
