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
	db   *sql.DB
	path string // 打开的数据库文件路径（备份用）
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
		sort INTEGER NOT NULL DEFAULT 0,
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
	// v2: 主题内标签 + 书签多对多关联
	`CREATE TABLE IF NOT EXISTS tag (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		theme_id INTEGER NOT NULL REFERENCES theme(id) ON DELETE CASCADE,
		name TEXT NOT NULL,
		color TEXT NOT NULL DEFAULT '',
		sort INTEGER NOT NULL DEFAULT 0,
		UNIQUE(theme_id, name)
	)`,
	`CREATE TABLE IF NOT EXISTS bookmark_tag (
		bookmark_id INTEGER NOT NULL REFERENCES bookmark(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
		PRIMARY KEY (bookmark_id, tag_id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_tag_theme ON tag(theme_id)`,
	`CREATE INDEX IF NOT EXISTS idx_bt_tag ON bookmark_tag(tag_id)`,
	// v4: 应用设置 KV（如书签打开方式偏好）
	`CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS idx_bookmark_theme ON bookmark(theme_id)`,
	`CREATE INDEX IF NOT EXISTS idx_account_theme ON account(theme_id)`,
	// v5: 账号类型/书签关联/账号标签（网站账号之外的设备、WiFi 密码等）
	`CREATE TABLE IF NOT EXISTS account_tag (
		account_id INTEGER NOT NULL REFERENCES account(id) ON DELETE CASCADE,
		tag_id INTEGER NOT NULL REFERENCES tag(id) ON DELETE CASCADE,
		PRIMARY KEY (account_id, tag_id)
	)`,
	`CREATE INDEX IF NOT EXISTS idx_at_tag ON account_tag(tag_id)`,
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
	s := &Store{db: db, path: path}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

// migrate 依次执行所有迁移语句；全部使用 IF NOT EXISTS 保证幂等。
// 注意：ALTER TABLE 不支持 IF NOT EXISTS，改用探测式迁移（见 ensureColumns）。
func (s *Store) migrate() error {
	for i, m := range migrations {
		if _, err := s.db.Exec(m); err != nil {
			return fmt.Errorf("store: 迁移 #%d 失败: %w", i+1, err)
		}
	}
	// v3: 老库补 bookmark.sort 列（新库建表语句已含）
	if err := s.ensureColumns(&columnPatch{
		table:  "bookmark",
		column: "sort",
		sql:    "ALTER TABLE bookmark ADD COLUMN sort INTEGER NOT NULL DEFAULT 0",
	}); err != nil {
		return err
	}
	// v5: 老库补账号类型与书签关联列（新库建表语句已含）
	if err := s.ensureColumns(
		&columnPatch{
			table:  "account",
			column: "kind",
			sql:    "ALTER TABLE account ADD COLUMN kind TEXT NOT NULL DEFAULT 'web'",
		},
		&columnPatch{
			table:  "account",
			column: "bookmark_id",
			sql:    "ALTER TABLE account ADD COLUMN bookmark_id INTEGER REFERENCES bookmark(id) ON DELETE SET NULL",
		},
	); err != nil {
		return err
	}
	return nil
}

// columnPatch 描述一次探测式列迁移。
type columnPatch struct {
	table  string
	column string
	sql    string
}

// ensureColumns 若表中缺少指定列则执行补列 SQL。
func (s *Store) ensureColumns(patches ...*columnPatch) error {
	for _, p := range patches {
		rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", p.table))
		if err != nil {
			return fmt.Errorf("store: 探测表 %s 结构失败: %w", p.table, err)
		}
		found := false
		for rows.Next() {
			var cid int
			var name, typ string
			var notNull, pk int
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
				rows.Close()
				return err
			}
			if name == p.column {
				found = true
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if !found {
			if _, err := s.db.Exec(p.sql); err != nil {
				return fmt.Errorf("store: 补列 %s.%s 失败: %w", p.table, p.column, err)
			}
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
