package favicon

import (
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// 缓存文件名白名单：16 位十六进制 + 已知图片扩展名，防目录穿越。
var cacheNameRe = regexp.MustCompile(`^[a-f0-9]{16}\.(ico|png|svg|jpg|webp)$`)

var contentTypeByExt = map[string]string{
	".ico":  "image/x-icon",
	".png":  "image/png",
	".svg":  "image/svg+xml",
	".jpg":  "image/jpeg",
	".webp": "image/webp",
}

// Handler 返回 Wails AssetServer 兜底处理器：服务 /favicons/<name> 缓存文件。
// 前端以同源相对路径访问，保证 canvas 绘制不被污染。
func Handler(c *Cache) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/favicons/") {
			http.NotFound(w, r)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/favicons/")
		if !cacheNameRe.MatchString(name) {
			http.NotFound(w, r)
			return
		}
		data, err := os.ReadFile(filepath.Join(c.dir, name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", contentTypeByExt[filepath.Ext(name)])
		w.Header().Set("Cache-Control", "max-age=86400")
		_, _ = w.Write(data)
	})
}
