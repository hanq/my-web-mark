package store

import (
	"database/sql"
	"fmt"
)

// GetSetting 读取 KV 设置；不存在返回空串。
func (s *Store) GetSetting(key string) (string, error) {
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("store: 读取设置失败: %w", err)
	}
	return v, nil
}

// SetSetting 写入/覆盖 KV 设置。
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value)
	if err != nil {
		return fmt.Errorf("store: 保存设置失败: %w", err)
	}
	return nil
}
