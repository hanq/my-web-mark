package app

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openBrowser 调用系统默认浏览器打开 URL。
func openBrowser(target string) error {
	switch runtime.GOOS {
	case "windows":
		if hasDangerousChars(target) {
			return fmt.Errorf("app: URL 含非法字符")
		}
		return execCommand("cmd", "/c", "start", "", target)
	case "darwin":
		return execCommand("open", target)
	default:
		return execCommand("xdg-open", target)
	}
}

// hasDangerousChars 拦截 shell 元字符，防 start 命令注入。
func hasDangerousChars(s string) bool {
	for _, r := range s {
		switch r {
		case '"', '\'', '`', '&', '|', ';', '<', '>', '$', '!', '\n', '\r':
			return true
		}
	}
	return false
}

// execCommand 执行外部命令（仅限内部可信参数），不等待退出。
func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = sysProcAttrHideWindow()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("app: 启动命令失败: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
