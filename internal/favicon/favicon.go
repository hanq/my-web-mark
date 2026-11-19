// Package favicon 负责网站图标（favicon）的抓取、磁盘缓存与 HTTP 分发。
//
// 抓取策略（依次尝试）：
//  1. https://<host>/favicon.ico
//  2. 解析 https://<host>/ 主页 HTML 中的 <link rel="...icon..." href>
//  3. http://<host>/favicon.ico 兜底
//
// 缓存文件按 host 哈希命名存于 <数据目录>/favicons/；失败记录负缓存
// （10 分钟）避免反复请求离线/慢站点。
package favicon

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	failTTL  = 10 * time.Minute
	maxBytes = 512 * 1024
	timeout  = 5 * time.Second
)

// Cache favicon 缓存器。
type Cache struct {
	dir    string
	client *http.Client

	mu   sync.Mutex
	ok   map[string]string    // host -> 缓存文件名
	fail map[string]time.Time // host -> 上次失败时间
}

// NewCache 用默认数据目录（%APPDATA%/my-web-mark/favicons）构造缓存。
func NewCache() *Cache {
	return NewCacheAt(defaultDir())
}

// NewCacheAt 用指定目录构造缓存（测试用）。
func NewCacheAt(dir string) *Cache {
	_ = os.MkdirAll(dir, 0o755)
	return &Cache{
		dir:    dir,
		client: &http.Client{Timeout: timeout},
		ok:     make(map[string]string),
		fail:   make(map[string]time.Time),
	}
}

func defaultDir() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = filepath.Join(home, "AppData", "Roaming")
		}
	}
	return filepath.Join(base, "my-web-mark", "favicons")
}

// Dir 返回缓存目录。
func (c *Cache) Dir() string { return c.dir }

// Ensure 确保指定 URL 站点的 favicon 已缓存，返回缓存文件名；失败返回空串。
func (c *Cache) Ensure(rawURL string) (string, error) {
	host := hostOf(rawURL)
	if host == "" {
		return "", fmt.Errorf("favicon: URL 缺少主机名")
	}

	c.mu.Lock()
	if name, hit := c.ok[host]; hit {
		c.mu.Unlock()
		return name, nil
	}
	if t, bad := c.fail[host]; bad && time.Since(t) < failTTL {
		c.mu.Unlock()
		return "", nil // 负缓存期内直接放弃
	}
	// 磁盘已存在（上次运行缓存）直接命中
	if existing, ok := findExisting(c.dir, host); ok {
		c.ok[host] = existing
		c.mu.Unlock()
		return existing, nil
	}
	c.mu.Unlock()

	data, ext := c.fetch(host)

	c.mu.Lock()
	defer c.mu.Unlock()
	if len(data) == 0 {
		c.fail[host] = time.Now()
		return "", nil
	}
	name := hashName(host) + ext
	if err := os.WriteFile(filepath.Join(c.dir, name), data, 0o644); err != nil {
		return "", fmt.Errorf("favicon: 写缓存失败: %w", err)
	}
	c.ok[host] = name
	return name, nil
}

// fetch 依次尝试各候选来源，返回图标字节与扩展名（空表示全部失败）。
func (c *Cache) fetch(host string) ([]byte, string) {
	https := "https://" + host
	candidates := []struct {
		url      string
		parseHTML bool
	}{
		{https + "/favicon.ico", false},
		{https + "/", true},
		{"http://" + host + "/favicon.ico", false},
	}
	for _, cand := range candidates {
		data, ext := c.get(cand.url, cand.parseHTML)
		if len(data) > 0 {
			return data, ext
		}
	}
	return nil, ""
}

// get 抓取单个 URL；parseHTML 为 true 时先取主页并解析 <link rel=icon>。
func (c *Cache) get(rawURL string, parseHTML bool) ([]byte, string) {
	resp, err := c.client.Get(rawURL)
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, ""
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes))
	if err != nil {
		return nil, ""
	}

	if parseHTML {
		href := extractIconHref(string(body), rawURL)
		if href == "" {
			return nil, ""
		}
		return c.get(href, false)
	}

	ext := sniffExt(resp.Header.Get("Content-Type"), body)
	if ext == "" {
		return nil, "" // 非图片内容（如错误页 HTML）拒收
	}
	return body, ext
}

// hostOf 提取 URL 主机（保留非默认端口，供构造请求 URL；文件名哈希同源使用）。
func hostOf(rawURL string) string {
	s := strings.TrimSpace(rawURL)
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	return u.Host
}

// findExisting 扫描缓存目录中该 host 的已存文件。
func findExisting(dir, host string) (string, bool) {
	prefix := hashName(host)
	matches, _ := filepath.Glob(filepath.Join(dir, prefix+".*"))
	if len(matches) > 0 {
		return filepath.Base(matches[0]), true
	}
	return "", false
}

func hashName(host string) string {
	sum := sha1.Sum([]byte(host))
	return hex.EncodeToString(sum[:])[:16]
}

// sniffExt 按 Content-Type 与字节魔数推断扩展名；非图片返回空。
func sniffExt(contentType string, data []byte) string {
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "image/png"):
		return ".png"
	case strings.Contains(ct, "image/svg+xml"):
		return ".svg"
	case strings.Contains(ct, "image/jpeg"):
		return ".jpg"
	case strings.Contains(ct, "image/webp"):
		return ".webp"
	case strings.Contains(ct, "image/x-icon"), strings.Contains(ct, "image/vnd.microsoft.icon"), strings.Contains(ct, "image/gif"):
		return ".ico"
	}
	// Content-Type 缺失/含糊时按魔数判断
	switch {
	case len(data) >= 8 && data[0] == 0x89 && data[1] == 'P' && data[2] == 'N':
		return ".png"
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8:
		return ".jpg"
	case len(data) >= 6 && (strings.HasPrefix(string(data), "<svg") || strings.Contains(string(data[:minInt(len(data), 200)]), "<svg")):
		return ".svg"
	case len(data) >= 4 && data[0] == 0 && data[1] == 0 && data[2] == 1 && data[3] == 0:
		return ".ico"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return ".webp"
	case len(data) >= 6 && string(data[:3]) == "GIF":
		return ".ico"
	}
	return ""
}

// --- HTML <link rel=icon> 解析 ---

var (
	linkTagRe = regexp.MustCompile(`(?i)<link[^>]*>`)
	relIconRe = regexp.MustCompile(`(?i)rel\s*=\s*["'][^"']*icon[^"']*["']`)
	hrefRe    = regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
)

// extractIconHref 从 HTML 中提取首个图标链接（相对地址基于 baseURL 补全）。
func extractIconHref(html, baseURL string) string {
	for _, tag := range linkTagRe.FindAllString(html, -1) {
		if !relIconRe.MatchString(tag) {
			continue
		}
		m := hrefRe.FindStringSubmatch(tag)
		if m == nil {
			continue
		}
		href := htmlUnescape(strings.TrimSpace(m[1]))
		if href == "" || strings.HasPrefix(href, "data:") {
			continue
		}
		base, err := url.Parse(baseURL)
		if err != nil {
			continue
		}
		ref, err := url.Parse(href)
		if err != nil {
			continue
		}
		return base.ResolveReference(ref).String()
	}
	return ""
}

func htmlUnescape(s string) string {
	r := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'")
	return r.Replace(s)
}

// minInt 返回较小值（兼容旧版本 Go 的内联实现）。
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
