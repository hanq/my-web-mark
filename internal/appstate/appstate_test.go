package appstate

import "testing"

// 主屏 1920x1080，副屏在右侧（1920,0,1920x1080）→ 虚拟屏 3840x1080
var vs = Rect{X: 0, Y: 0, W: 3840, H: 1080}
var ps = Rect{X: 0, Y: 0, W: 1920, H: 1080}

func TestSanitizeNormalMultiScreen(t *testing.T) {
	// 副屏上的窗口（x > 1920）原样保留
	in := WindowState{X: 2200, Y: 120, W: 1280, H: 800}
	out := Sanitize(in, vs, ps, 1024, 640, 1280, 800)
	if out != in {
		t.Fatalf("副屏合法状态不应被修改: %+v", out)
	}
	// 左侧负坐标副屏：虚拟屏含负坐标，窗口原样保留
	vsNeg := Rect{X: -1920, Y: 0, W: 3840, H: 1080}
	in = WindowState{X: -1800, Y: 100, W: 1280, H: 800}
	out = Sanitize(in, vsNeg, ps, 1024, 640, 1280, 800)
	if out != in {
		t.Fatalf("负坐标屏合法状态不应被修改: %+v", out)
	}
	// 左/上越界（大交集）：clamp 回虚拟屏边界
	in = WindowState{X: -2500, Y: -100, W: 1280, H: 800}
	out = Sanitize(in, vsNeg, ps, 1024, 640, 1280, 800)
	if out.X != -1920 || out.Y != 0 {
		t.Fatalf("负向越界应 clamp 到虚拟屏边界: %+v", out)
	}
}

func TestSanitizeFallbackWhenScreenGone(t *testing.T) {
	// 副屏断开：虚拟屏缩回主屏，窗口仍在旧副屏位置 → 交集为 0 → 主屏居中
	gone := Rect{X: 0, Y: 0, W: 1920, H: 1080}
	in := WindowState{X: 2200, Y: 120, W: 1280, H: 800}
	out := Sanitize(in, gone, ps, 1024, 640, 1280, 800)
	if out.X != (1920-1280)/2 || out.Y != (1080-800)/3 {
		t.Fatalf("应降级主屏居中: %+v", out)
	}
	// 仅露出 50px（< 阈值 200）也降级
	in = WindowState{X: 1800, Y: 0, W: 1280, H: 800}
	out = Sanitize(in, gone, ps, 1024, 640, 1280, 800)
	if out.X != (1920-1280)/2 {
		t.Fatalf("交集不足应降级: %+v", out)
	}
}

func TestSanitizeClampWhenResolutionShrinks(t *testing.T) {
	// 分辨率降为 1080p：跨屏窗口大交集属合法状态，仅验证不降级不崩溃
	small := Rect{X: 0, Y: 0, W: 1920, H: 1080}
	in := WindowState{X: 1500, Y: 900, W: 1600, H: 900}
	out := Sanitize(in, small, ps, 1024, 640, 1280, 800)
	if out.X != 1500 || out.Y != 900 {
		t.Fatalf("大交集跨屏窗口应保留位置: %+v", out)
	}
	// 尺寸超屏 → 收敛到屏宽
	in = WindowState{X: 0, Y: 0, W: 3000, H: 1300}
	out = Sanitize(in, small, ps, 1024, 640, 1280, 800)
	if out.W > small.W || out.H > small.H {
		t.Fatalf("尺寸未收敛: %+v", out)
	}
	// 尺寸小于最小值 → 提升
	in = WindowState{X: 0, Y: 0, W: 400, H: 300}
	out = Sanitize(in, small, ps, 1024, 640, 1280, 800)
	if out.W < 1024 || out.H < 640 {
		t.Fatalf("最小尺寸未保证: %+v", out)
	}
}
