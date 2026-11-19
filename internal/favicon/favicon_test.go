package favicon

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractIconHref(t *testing.T) {
	html := `<!doctype html><html><head>
		<link rel="stylesheet" href="/style.css">
		<link rel="shortcut icon" href="/static/favicon.ico?v=2&amp;x=1">
		<link rel="apple-touch-icon" href="/touch.png">
		</head></html>`
	got := extractIconHref(html, "https://example.com/home/")
	if got != "https://example.com/static/favicon.ico?v=2&x=1" {
		t.Fatalf("解析结果错误: %q", got)
	}
	// 相对路径补全
	got = extractIconHref(`<link rel="icon" href="img/fav.png">`, "https://a.com/b/")
	if got != "https://a.com/b/img/fav.png" {
		t.Fatalf("相对路径补全错误: %q", got)
	}
	// data URI 跳过、无图标返回空
	if extractIconHref(`<link rel="icon" href="data:image/png;base64,xxx">`, "https://a.com") != "" {
		t.Fatal("data URI 应被跳过")
	}
	if extractIconHref(`<p>no icon</p>`, "https://a.com") != "" {
		t.Fatal("无图标应返回空")
	}
}

func TestSniffExt(t *testing.T) {
	png := append([]byte{0x89, 'P', 'N', 'G'}, make([]byte, 10)...)
	if got := sniffExt("", png); got != ".png" {
		t.Fatalf("PNG 魔数识别失败: %q", got)
	}
	if got := sniffExt("image/svg+xml", []byte("<svg/>")); got != ".svg" {
		t.Fatalf("svg content-type 识别失败: %q", got)
	}
	if got := sniffExt("text/html", []byte("<html>error page</html>")); got != "" {
		t.Fatalf("HTML 应被拒绝: %q", got)
	}
}

func TestEnsureWithServer(t *testing.T) {
	png := append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 32)...)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(png)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	dir := t.TempDir()
	c := NewCacheAt(dir)

	name, err := c.Ensure(srv.URL)
	if err != nil || name == "" {
		t.Fatalf("Ensure 失败: %v name=%q", err, name)
	}
	if !strings.HasSuffix(name, ".png") {
		t.Fatalf("扩展名应为 .png: %q", name)
	}
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatalf("缓存文件未写入: %v", err)
	}

	// 第二次命中内存缓存
	name2, err := c.Ensure(srv.URL)
	if err != nil || name2 != name {
		t.Fatalf("二次 Ensure 不一致: %q vs %q (%v)", name2, name, err)
	}

	// 失败站点进入负缓存（返回空串而非错误）
	name3, _ := c.Ensure("https://nonexistent.invalid")
	if name3 != "" {
		t.Fatalf("不可达站点应返回空: %q", name3)
	}
}

func TestHandlerSecurity(t *testing.T) {
	c := NewCacheAt(t.TempDir())
	h := Handler(c)

	// 目录穿越拒绝
	req := httptest.NewRequest("GET", "/favicons/../../../windows/system32/config", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("目录穿越应 404: %d", rec.Code)
	}
	// 非法文件名拒绝
	req = httptest.NewRequest("GET", "/favicons/zzz.png", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("非法文件名应 404: %d", rec.Code)
	}
}

func TestHostOf(t *testing.T) {
	cases := map[string]string{
		"https://github.com/a":       "github.com",
		"github.com":                 "github.com",
		"http://localhost:3000/x":    "localhost:3000",
		"":                           "",
	}
	for in, want := range cases {
		if got := hostOf(in); got != want {
			t.Fatalf("hostOf(%q)=%q 期望 %q", in, got, want)
		}
	}
}
