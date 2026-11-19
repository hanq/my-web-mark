// Package appstate 应用运行状态持久化：窗口位置/大小/最大化状态的
// 保存、加载与多屏环境下的合法性修正。
package appstate

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// WindowState 窗口状态快照（虚拟化逻辑坐标，与 DPI 无关）。
type WindowState struct {
	X         int  `json:"x"`
	Y         int  `json:"y"`
	W         int  `json:"w"`
	H         int  `json:"h"`
	Maximised bool `json:"maximised"`
}

// Rect 屏幕矩形。
type Rect struct {
	X, Y, W, H int
}

// Right 右边界。
func (r Rect) Right() int { return r.X + r.W }

// Bottom 下边界。
func (r Rect) Bottom() int { return r.Y + r.H }

// StatePath 状态文件路径。
func StatePath() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, "AppData", "Roaming")
		}
	}
	return filepath.Join(base, "my-web-mark", "window-state.json")
}

// Load 读取窗口状态；文件不存在或损坏返回 (nil, nil)，由调用方走默认值。
func Load() (*WindowState, error) {
	raw, err := os.ReadFile(StatePath())
	if err != nil {
		return nil, nil // 首次运行
	}
	var s WindowState
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, fmt.Errorf("appstate: 状态文件损坏: %w", err)
	}
	// 最大化快照允许 rect 为 0（恢复时直接最大化，还原尺寸交给系统）
	if !s.Maximised && (s.W <= 0 || s.H <= 0) {
		return nil, nil
	}
	return &s, nil
}

// Save 原子保存窗口状态（临时文件 + 重命名，避免写一半崩溃损坏）。
func Save(s WindowState) error {
	path := StatePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("appstate: 创建目录失败: %w", err)
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("appstate: 写临时文件失败: %w", err)
	}
	return os.Rename(tmp, path)
}

// 可见性阈值：窗口至少露出这么大面积才认为"还在屏幕上"（保证标题栏可拖回）。
const (
	minVisibleW = 200
	minVisibleH = 150
)

// Sanitize 校验并修正窗口状态，覆盖多屏场景的降级策略：
//
//  1. 尺寸先收敛：不小于最小值，不超过虚拟桌面；
//  2. 位置与虚拟桌面交集不足（副屏断开、投影模式切换、分辨率大幅降低）
//     → 降级为主屏居中（主屏永远存在）；
//  3. 交集足够但越界 → clamp 回可见区域（保持多屏负坐标原样恢复）。
func Sanitize(s WindowState, virtual, primary Rect, minW, minH, defW, defH int) WindowState {
	out := s

	// 1) 尺寸收敛
	if out.W < minW {
		out.W = minW
	}
	if out.H < minH {
		out.H = minH
	}
	if out.W > virtual.W {
		out.W = virtual.W
	}
	if out.H > virtual.H {
		out.H = virtual.H
	}

	// 2) 交集判定：不足阈值整体降级主屏居中
	ix := minInt(out.X+out.W, virtual.Right()) - maxInt(out.X, virtual.X)
	iy := minInt(out.Y+out.H, virtual.Bottom()) - maxInt(out.Y, virtual.Y)
	if ix < minVisibleW || iy < minVisibleH {
		out.X = primary.X + (primary.W-defW)/2
		out.Y = primary.Y + (primary.H-defH)/3
		if out.X < primary.X {
			out.X = primary.X
		}
		if out.Y < primary.Y {
			out.Y = primary.Y
		}
		return out
	}

	// 3) clamp 回虚拟屏（留出阈值面积可见）
	out.X = clampInt(out.X, virtual.X, virtual.Right()-minVisibleW)
	out.Y = clampInt(out.Y, virtual.Y, virtual.Bottom()-minVisibleH)
	return out
}

func clampInt(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
