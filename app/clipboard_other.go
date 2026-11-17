//go:build !windows

package app

import (
	"fmt"
	"os/exec"
	"syscall"
)

// sysProcAttrHideWindow 非 Windows 平台无需隐藏窗口。
func sysProcAttrHideWindow() *syscall.SysProcAttr { return nil }

// copyToClipboard 非 Windows 平台剪贴板写入（macOS/Linux 桌面）。
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	if text == "" {
		return nil // 清空操作在非 Windows 平台略过
	}
	if _, err := exec.LookPath("pbcopy"); err == nil {
		cmd = exec.Command("pbcopy")
	} else if _, err := exec.LookPath("wl-copy"); err == nil {
		cmd = exec.Command("wl-copy")
	} else if _, err := exec.LookPath("xclip"); err == nil {
		cmd = exec.Command("xclip", "-selection", "clipboard")
	} else {
		return fmt.Errorf("app: 未找到可用剪贴板工具")
	}
	cmd.Stdin = bytes.NewBufferString(text)
	return cmd.Run()
}
