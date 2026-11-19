//go:build !windows

package appstate

// VirtualScreen 非 Windows 平台的占位实现（桌面应用当前仅发布 Windows）。
func VirtualScreen() Rect { return Rect{X: 0, Y: 0, W: 1920, H: 1080} }

// PrimaryScreen 非 Windows 平台的占位实现。
func PrimaryScreen() Rect { return Rect{X: 0, Y: 0, W: 1920, H: 1080} }
