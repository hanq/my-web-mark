package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Backup 备份数据库到 <db目录>/backup/，文件名含时间戳。
// 使用 VACUUM INTO 生成一致性快照（WAL 模式下也能拿到完整数据），
// 保留最近 keep 份，超出按文件名（即时间）删除旧的。
func (s *Store) Backup(keep int) error {
	if s.db == nil {
		return fmt.Errorf("store: 数据库未打开")
	}
	if keep <= 0 {
		keep = 5
	}
	backupDir := filepath.Join(filepath.Dir(s.path), "backup")
	if err := os.MkdirAll(backupDir, 0o755); err != nil {
		return fmt.Errorf("store: 创建备份目录失败: %w", err)
	}
	dest := filepath.Join(backupDir, fmt.Sprintf("mywebmark-%s.db", time.Now().Format("20060102-150405.000"))) // 含毫秒防同秒覆盖
	// 目标文件必须不存在（VACUUM INTO 要求）
	_ = os.Remove(dest)
	if _, err := s.db.Exec(fmt.Sprintf(`VACUUM INTO '%s'`, strings.ReplaceAll(dest, "'", "''"))); err != nil {
		return fmt.Errorf("store: 备份失败: %w", err)
	}
	return pruneBackups(backupDir, keep)
}

// pruneBackups 删除超出保留数量的旧备份。
func pruneBackups(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), "mywebmark-") && strings.HasSuffix(e.Name(), ".db") {
			names = append(names, e.Name())
		}
	}
	// 文件名含时间戳，字典序即时间序
	sort.Strings(names)
	for len(names) > keep {
		oldest := filepath.Join(dir, names[0])
		if err := os.Remove(oldest); err != nil {
			return err
		}
		names = names[1:]
	}
	return nil
}
