package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// BrowserService 拖拽/外部 URL 处理辅助。
type BrowserService struct {
	sh *shared
}

// ParsedURL 拖拽进来的 URL 解析结果。
type ParsedURL struct {
	RawURL string `json:"rawUrl"`
	Title  string `json:"title"` // 可能为空（前端可再调 FetchTitle）
	Host   string `json:"host"`
}

// ParseDroppedURL 解析拖拽进来的文本，提取首个合法 URL。
// dataTransfer 可能是 text/uri-list、text/html 或纯文本，这里统一容错处理。
func (b *BrowserService) ParseDroppedURL(text string) (*ParsedURL, error) {
	u := extractURL(text)
	if u == "" {
		return nil, fmt.Errorf("app: 未在拖入内容中找到 URL")
	}
	pu, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("app: URL 格式非法: %w", err)
	}
	if pu.Scheme == "" {
		pu.Scheme = "https"
		u = pu.String()
	}
	if pu.Host == "" {
		return nil, fmt.Errorf("app: URL 缺少主机名")
	}
	title := pu.Host
	// 去掉常见端口号展示噪音
	if i := strings.Index(title, ":"); i >= 0 {
		title = title[:i]
	}
	return &ParsedURL{RawURL: u, Title: title, Host: pu.Host}, nil
}

// extractURL 从多行/HTML 文本中提取首个 http(s) URL。
func extractURL(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	// text/uri-list：# 开头是注释
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "http://") || strings.HasPrefix(line, "https://") {
			return strings.TrimRight(line, "\r")
		}
	}
	// HTML 片段里找 href
	if m := hrefRe.FindStringSubmatch(text); m != nil {
		return htmlUnescape(m[1])
	}
	// 兜底：裸域名（补协议后再返回）
	if m := bareHostRe.FindStringSubmatch(text); m != nil {
		return "https://" + m[0]
	}
	return ""
}

var (
	hrefRe     = regexp.MustCompile(`href=["']([^"']+)["']`)
	bareHostRe = regexp.MustCompile(`^(?:[a-z0-9-]+\.)+[a-z]{2,}(?:/[^\s]*)?`)
)

// htmlUnescape 最小化 HTML 反转义。
func htmlUnescape(s string) string {
	r := strings.NewReplacer("&amp;", "&", "&lt;", "<", "&gt;", ">", "&quot;", "\"", "&#39;", "'")
	return r.Replace(s)
}

// PageTitleService 抓取网页标题（导入书签时补全标题用）。
type PageTitleService struct{}

var titleRe = regexp.MustCompile(`(?i)<title[^>]*>([^<]*)</title>`)

// FetchTitle 抓取 URL 页面的 <title>；超时 8s，失败返回空串而非错误。
func (p *PageTitleService) FetchTitle(target string) (string, error) {
	if !strings.Contains(target, "://") {
		target = "https://" + target
	}
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("app: 仅支持 http(s) URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) my-web-mark/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", nil // 抓取失败不阻塞导入
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil
	}
	// 只读前 64KB 足够拿到 title
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return "", nil
	}
	if m := titleRe.FindSubmatch(body); m != nil {
		return strings.TrimSpace(decodeHTMLText(string(m[1]))), nil
	}
	return "", nil
}

// decodeHTMLText 处理 charset 常见场景（UTF-8 为主，GBK 粗略回退）。
func decodeHTMLText(s string) string {
	// WebView 前端主要面对 UTF-8；此处仅做基本反转义
	return htmlUnescape(s)
}

// OpenURLInput 用系统默认浏览器打开 URL 的入参。
type OpenURLInput struct {
	URL string `json:"url"`
}

// OpenInDefaultBrowser 用系统默认浏览器打开书签（前端不能直接 window.open 外链）。
func (b *BrowserService) OpenInDefaultBrowser(in OpenURLInput) error {
	u := strings.TrimSpace(in.URL)
	if u == "" {
		return fmt.Errorf("app: URL 为空")
	}
	if !strings.Contains(u, "://") {
		u = "https://" + u
	}
	return openBrowser(u)
}
