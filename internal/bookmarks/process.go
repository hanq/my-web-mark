package bookmarks

import (
	"fmt"
	"os"
	"strings"

	"github.com/shirou/gopsutil/v3/process"
)

// DetectRunning 返回当前正在运行的浏览器列表。
// 进程名做小写精确匹配，避免误判。
func DetectRunning() ([]BrowserKind, error) {
	processes, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("bookmarks: 枚举进程失败: %w", err)
	}
	running := make(map[BrowserKind]bool)
	for _, p := range processes {
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}
		lower := strings.ToLower(name)
		for kind, names := range ProcessNames {
			for _, n := range names {
				if lower == n {
					running[kind] = true
				}
			}
		}
	}
	var out []BrowserKind
	for _, kind := range AllBrowsers {
		if running[kind] {
			out = append(out, kind)
		}
	}
	return out, nil
}

// AppDirs 汇总应用数据目录（供上层注入测试，生产用真实环境变量）。
type AppDirs struct {
	AppData      string // %APPDATA%
	LocalAppData string // %LOCALAPPDATA%
}

// DefaultAppDirs 从环境变量读取标准目录。
func DefaultAppDirs() AppDirs {
	return AppDirs{
		AppData:      os.Getenv("APPDATA"),
		LocalAppData: os.Getenv("LOCALAPPDATA"),
	}
}
