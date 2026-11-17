//go:build windows

package app

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
)

// createNoWindow 防止 cmd/PowerShell 弹出控制台窗口。
const createNoWindow = 0x08000000

// sysProcAttrHideWindow Windows 下隐藏子进程控制台窗口。
func sysProcAttrHideWindow() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// copyToClipboard 通过 PowerShell 管道写剪贴板；空串即清空。
func copyToClipboard(text string) error {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		"$input | Set-Clipboard")
	cmd.Stdin = bytes.NewBufferString(text)
	cmd.SysProcAttr = sysProcAttrHideWindow()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("app: 写剪贴板失败: %w (%s)", err, stderr.String())
	}
	return nil
}
