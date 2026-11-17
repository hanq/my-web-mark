// Package store 提供 SQLite 数据库的初始化、迁移与访问。
// 数据库文件默认位于 %APPDATA%/my-web-mark/mywebmark.db。
package store

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// Store 封装数据库连接与常用操作。
type Store struct {
	db *sql.DB
}

// migrations 按序执行的建表/变更语句；新增表结构时追加条目即可。
var migrations = []string{
	// v1: 初始表结构
	`CREATE TABLE IF NOT EXISTS theme (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		color TEXT NOT NULL DEFAULT '#4f8cff',
		icon TEXT DEFAULT '',
		sort INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS bookmark (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		theme_id INTEGER NOT NULL REFERENCES theme(id) ON DELETE CASCADE,
		title TEXT NOT NULL,
		url TEXT NOT NULL,
		favicon TEXT DEFAULT '',
		note TEXT DEFAULT '',
		pinned INTEGER NOT NULL DEFAULT 0,
		source TEXT NOT NULL DEFAULT 'manual',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(theme_id, url)
	)`,
	`CREATE TABLE IF NOT EXISTS account (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		theme_id INTEGER NOT NULL REFERENCES theme(id) ON DELETE CASCADE,
		site TEXT NOT NULL,
		username TEXT NOT NULL,
		password_cipher BLOB NOT NULL,
		url TEXT DEFAULT '',
		note TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS vault_meta (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		salt BLOB NOT NULL,
		password_hash BLOB NOT NULL,
		kdf_params TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_bookmark_theme ON bookmark(theme_id)`,
	`CREATE INDEX IF NOT EXISTS idx_account_theme ON account(theme_id)`,
}

// DefaultDBPath 返回默认数据库文件路径（%APPDATA%/my-web-mark/mywebmark.db），
// 目录不存在时自动创建。
func DefaultDBPath() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("store: 无法定位用户目录: %w", err)
		}
		appData = filepath.Join(home, "AppData", "Roaming")
	}
	dir := filepath.Join(appData, "my-web-mark")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("store: 创建数据目录失败: %w", err)
	}
	return filepath.Join(dir, "mywebmark.db"), nil
}

// Open 打开（必要时创建）数据库并执行迁移。
func Open(path string) (*Store, error) {
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("store: 打开数据库失败: %w", err)
	}
	// modernc/sqlite 单写连接避免 database locked
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// migrate 依次执行所有迁移语句；全部使用 IF NOT EXISTS 保证幂等。
func (s *Store) migrate() error {
	for i, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("store: 迁移 #%d 失败: %w", i+1, err)
		}
	}
	return nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB 暴露底层连接供各 DAO 使用。
func (s *Store) DB() *sql.DB {
	if s.db == nil {
		return nil
	}
	return s.db
}

// ErrNotFound 表示目标记录不存在。
var ErrNotFound = errors.New("store: 记录不存在")
