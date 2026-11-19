//go:build windows

package appstate

import "syscall"

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetSystemMetrics = user32.NewProc("GetSystemMetrics")
)

const (
	smCXScreen        = 0 // 主屏宽
	smCYScreen        = 1 // 主屏高
	smXVirtualScreen  = 76 // 虚拟桌面左上 x（副屏可为负）
	smYVirtualScreen  = 77 // 虚拟桌面左上 y
	smCXVirtualScreen = 78 // 虚拟桌面总宽
	smCYVirtualScreen = 79 // 虚拟桌面总高
)

func sysMetrics(index int) int {
	r, _, _ := procGetSystemMetrics.Call(uintptr(index))
	return int(r)
}

// VirtualScreen 返回虚拟桌面外接矩形（所有显示器覆盖区域）。
// Go 默认 DPI unaware：坐标为系统虚拟化的 96dpi 逻辑坐标，
// 与 Wails 窗口 API 的坐标系同源自洽，无需换算。
func VirtualScreen() Rect {
	x := sysMetrics(smXVirtualScreen)
	y := sysMetrics(smYVirtualScreen)
	w := sysMetrics(smCXVirtualScreen)
	h := sysMetrics(smCYVirtualScreen)
	return Rect{X: x, Y: y, W: w, H: h}
}

// PrimaryScreen 返回主显示器矩形（原点 0,0）。
func PrimaryScreen() Rect {
	return Rect{X: 0, Y: 0, W: sysMetrics(smCXScreen), H: sysMetrics(smCYScreen)}
}
