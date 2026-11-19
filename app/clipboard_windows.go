//go:build windows

package app

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

// createNoWindow 防止 cmd/PowerShell 弹出控制台窗口。
const createNoWindow = 0x08000000

// sysProcAttrHideWindow Windows 下隐藏子进程控制台窗口。
func sysProcAttrHideWindow() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}

// 剪贴板相关 Win32 常量。
const (
	cfUnicodeText = 13     // 剪贴板格式：UTF-16LE 文本
	gmemMoveable  = 0x0002 // GlobalAlloc 可移动内存
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")
	procGlobalAlloc      = kernel32.NewProc("GlobalAlloc")
	procGlobalLock       = kernel32.NewProc("GlobalLock")
	procGlobalUnlock     = kernel32.NewProc("GlobalUnlock")
	procGlobalFree       = kernel32.NewProc("GlobalFree")
	procRtlMoveMemory    = kernel32.NewProc("RtlMoveMemory")
)

// copyToClipboard 将文本写入系统剪贴板；空串表示清空。
//
// 直接调用 Win32 CF_UNICODETEXT，避免经由 PowerShell 标准输入时的编码转换：
// 管道内容按控制台代码页（中文 Windows 为 GBK）解码，会把 UTF-8 中文变成乱码
// （如「名称：」显示为「鍚嶇О锛�」）。原生写入不经过任何编码转换。
// 原生调用失败时退回 PowerShell，但用 -EncodedCommand + Base64 传递文本，
// 全程显式 UTF-8/UTF-16 转换，不依赖控制台代码页。
func copyToClipboard(text string) error {
	err := writeClipboardUnicode(text)
	if err == nil {
		return nil
	}
	if text == "" {
		return err
	}
	if fallbackErr := writeClipboardPowerShell(text); fallbackErr != nil {
		return fmt.Errorf("%w；兜底写入也失败: %v", err, fallbackErr)
	}
	return nil
}

// writeClipboardUnicode 用 Win32 API 写入 Unicode 文本；空串执行清空。
func writeClipboardUnicode(text string) error {
	if err := openClipboard(); err != nil {
		return err
	}
	defer procCloseClipboard.Call()

	if text == "" {
		procEmptyClipboard.Call()
		return nil
	}

	// CF_UNICODETEXT 要求 UTF-16LE 且以 0 结尾，所有权转移给系统。
	buf, err := syscall.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("app: 文本包含非法字符: %w", err)
	}
	size := uintptr(len(buf)) * unsafe.Sizeof(uint16(0))
	handle, _, callErr := procGlobalAlloc.Call(gmemMoveable, size)
	if handle == 0 {
		return fmt.Errorf("app: 分配剪贴板内存失败: %v", callErr)
	}
	ptr, _, _ := procGlobalLock.Call(handle)
	if ptr == 0 {
		procGlobalFree.Call(handle)
		return fmt.Errorf("app: 锁定剪贴板内存失败")
	}
	// 用 RtlMoveMemory 复制，避免 uintptr→unsafe.Pointer 的转换（go vet 会告警）
	src := uintptr(unsafe.Pointer(&buf[0]))
	procRtlMoveMemory.Call(ptr, src, size)
	procGlobalUnlock.Call(handle)

	procEmptyClipboard.Call()
	if ok, _, callErr := procSetClipboardData.Call(cfUnicodeText, handle); ok == 0 {
		// 交给系统失败时必须自行释放，否则泄漏
		procGlobalFree.Call(handle)
		return fmt.Errorf("app: 写入剪贴板失败: %v", callErr)
	}
	return nil
}

// openClipboard 打开剪贴板；被其他程序短暂占用时重试。
func openClipboard() error {
	var lastErr error
	for i := 0; i < 10; i++ {
		if ok, _, callErr := procOpenClipboard.Call(0); ok != 0 {
			return nil
		} else {
			lastErr = callErr
		}
		time.Sleep(20 * time.Millisecond)
	}
	return fmt.Errorf("app: 打开剪贴板失败（可能被其他程序占用）: %v", lastErr)
}

// writeClipboardPowerShell 兜底写入：脚本与文本都用 Base64 传递，不经过控制台代码页。
func writeClipboardPowerShell(text string) error {
	payload := base64.StdEncoding.EncodeToString([]byte(text))
	script := "[Console]::OutputEncoding=[Text.Encoding]::UTF8; " +
		"$t=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String('" + payload + "')); " +
		"Set-Clipboard -Value $t"
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive",
		"-EncodedCommand", encodeCommand(script))
	cmd.SysProcAttr = sysProcAttrHideWindow()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("app: 写剪贴板失败: %w (%s)", err, stderr.String())
	}
	return nil
}

// encodeCommand 按 PowerShell -EncodedCommand 要求编码为 Base64(UTF-16LE)。
func encodeCommand(script string) string {
	units := utf16.Encode([]rune(script))
	raw := make([]byte, len(units)*2)
	for i, u := range units {
		raw[i*2] = byte(u)
		raw[i*2+1] = byte(u >> 8)
	}
	return base64.StdEncoding.EncodeToString(raw)
}
