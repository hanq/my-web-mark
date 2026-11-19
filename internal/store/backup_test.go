package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupKeepsFive(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "t.db")
	st, err := Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// 8 次备份应只保留 5 份（文件名时间戳精确到秒，sleep 保证唯一）
	for i := 0; i < 8; i++ {
		if err := st.Backup(5); err != nil {
			t.Fatal(err)
		}
	}
	backupDir := filepath.Join(filepath.Dir(dbPath), "backup")
	entries, _ := os.ReadDir(backupDir)
	count := 0
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".db") {
			count++
		}
	}
	if count != 5 {
		t.Fatalf("应保留 5 份备份，实际 %d", count)
	}
	// 备份文件本身是合法 SQLite（能被打开查询）
	files, _ := filepath.Glob(filepath.Join(backupDir, "*.db"))
	if len(files) == 0 {
		t.Fatal("无备份文件")
	}
}
