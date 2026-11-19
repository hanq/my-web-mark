package app

import (
	"my-web-mark/internal/favicon"
)

// FaviconService 网站图标抓取与缓存（供前端卡片徽标显示）。
// 返回值为缓存文件名；前端以 /favicons/<name> 同源路径访问。
type FaviconService struct {
	cache *favicon.Cache
}

// Ensure 确保站点 favicon 已缓存；成功返回文件名，失败返回空串（前端回退首字母）。
func (f *FaviconService) Ensure(rawURL string) (string, error) {
	return f.cache.Ensure(rawURL)
}
