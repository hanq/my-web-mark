package bookmarks

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"

	xhtml "golang.org/x/net/html"
)

// ExportBookmark 主题书签导出的中间表示。
// Folder 供 HTML 归组（取首标签）；Tags 供 JSON 保留完整标签。
type ExportBookmark struct {
	Title  string   `json:"title"`
	URL    string   `json:"url"`
	Note   string   `json:"note,omitempty"`
	Folder string   `json:"folder,omitempty"`
	Pinned bool     `json:"pinned,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

// ExportThemeFile JSON 导出文件结构（format 字段用于导入时自动检测）。
type ExportThemeFile struct {
	Format    string          `json:"format"`
	Version   int             `json:"version"`
	Theme     ExportThemeMeta `json:"theme"`
	Bookmarks []ExportBookmark `json:"bookmarks"`
}

// ExportThemeMeta 主题元信息。
type ExportThemeMeta struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

const exportFormatID = "my-web-mark-theme"

// RenderNetscapeHTML 渲染为 Netscape 书签 HTML（浏览器可直接导入）。
// 标签映射为子文件夹；HTML 格式不携带置顶信息（格式限制）。
func RenderNetscapeHTML(themeName string, items []ExportBookmark) string {
	var sb strings.Builder
	sb.WriteString("<!DOCTYPE NETSCAPE-Bookmark-file-1>\n")
	sb.WriteString("<!-- This is an automatically generated file.\n     It will be read and overwritten.\n     DO NOT EDIT! -->\n")
	sb.WriteString("<META HTTP-EQUIV=\"Content-Type\" CONTENT=\"text/html; charset=UTF-8\">\n")
	sb.WriteString("<TITLE>Bookmarks</TITLE>\n")
	sb.WriteString("<H1>" + html.EscapeString(themeName) + "</H1>\n")
	sb.WriteString("<DL><p>\n")

	// 按首标签归组（保持出现顺序）
	var order []string
	grouped := map[string][]ExportBookmark{}
	var loose []ExportBookmark
	for _, it := range items {
		if it.Folder == "" {
			loose = append(loose, it)
			continue
		}
		if _, ok := grouped[it.Folder]; !ok {
			order = append(order, it.Folder)
		}
		grouped[it.Folder] = append(grouped[it.Folder], it)
	}

	writeItem := func(it ExportBookmark, indent string) {
		sb.WriteString(indent + "<DT><A HREF=\"" + html.EscapeString(it.URL) + "\">" + html.EscapeString(it.Title) + "</A>\n")
		if it.Note != "" {
			sb.WriteString(indent + "<DD>" + html.EscapeString(it.Note) + "\n")
		}
	}
	for _, f := range order {
		sb.WriteString("    <DT><H3>" + html.EscapeString(f) + "</H3>\n    <DL><p>\n")
		for _, it := range grouped[f] {
			writeItem(it, "        ")
		}
		sb.WriteString("    </DL><p>\n")
	}
	for _, it := range loose {
		writeItem(it, "    ")
	}
	sb.WriteString("</DL><p>\n")
	return sb.String()
}

// MarshalThemeJSON 序列化 JSON 导出内容（完整保留标签/备注/置顶）。
func MarshalThemeFile(meta ExportThemeMeta, items []ExportBookmark) ([]byte, error) {
	f := ExportThemeFile{
		Format:    exportFormatID,
		Version:   1,
		Theme:     meta,
		Bookmarks: items,
	}
	return json.MarshalIndent(f, "", "  ")
}

// ParseThemeFile 解析 JSON 导出内容。
func ParseThemeFile(data []byte) (*ExportThemeFile, error) {
	var f ExportThemeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("bookmarks: JSON 解析失败: %w", err)
	}
	if f.Format != exportFormatID {
		return nil, fmt.Errorf("bookmarks: 不是 %s 格式文件", exportFormatID)
	}
	return &f, nil
}

// DetectExportFormat 自动检测内容格式：json 或 html。
func DetectExportFormat(content string) string {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, "{") && strings.Contains(trimmed, exportFormatID) {
		return "json"
	}
	return "html"
}

// ParseNetscapeHTML 解析 Netscape 书签 HTML，返回主题名（H1）与扁平书签。
// 标签文件夹映射回 Folder 字段；<DD> 文本映射为备注。
func ParseNetscapeHTML(content string) (string, []ExportBookmark, error) {
	doc, err := xhtml.Parse(strings.NewReader(content))
	if err != nil {
		return "", nil, fmt.Errorf("bookmarks: HTML 解析失败: %w", err)
	}

	var themeName string
	var items []ExportBookmark

	var walk func(n *xhtml.Node)
	walk = func(n *xhtml.Node) {
		if n.Type == xhtml.ElementNode {
			switch n.Data {
			case "h1":
				if themeName == "" {
					themeName = nodeText(n)
				}
			case "a":
				href := ""
				for _, attr := range n.Attr {
					if strings.EqualFold(attr.Key, "href") {
						href = attr.Val
						break
					}
				}
				if strings.HasPrefix(strings.ToLower(href), "http") {
					items = append(items, ExportBookmark{
						Title:  nodeText(n),
						URL:    href,
						Note:   noteOf(n),
						Folder: folderOf(n),
					})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return strings.TrimSpace(themeName), items, nil
}

// nodeText 拼接节点内全部文本（已由解析器解码实体）。
func nodeText(n *xhtml.Node) string {
	var sb strings.Builder
	var rec func(*xhtml.Node)
	rec = func(x *xhtml.Node) {
		if x.Type == xhtml.TextNode {
			sb.WriteString(x.Data)
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			rec(c)
		}
	}
	rec(n)
	return strings.TrimSpace(sb.String())
}

// firstByTag 在子树中找第一个指定标签元素。
func firstByTag(n *xhtml.Node, tag string) *xhtml.Node {
	if n.Type == xhtml.ElementNode && n.Data == tag {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := firstByTag(c, tag); found != nil {
			return found
		}
	}
	return nil
}

// folderOf 书签所属文件夹。兼容 HTML5 解析器产出的两种结构：
//  1. dt 嵌套变体（Chrome/FF 导出经解析后常见）：<dt><h3>标题</h3><dl>…书签…</dl></dt>，
//     即书签的祖先 DT 同时包含 H3 与 DL；
//  2. 兄弟变体：dl 与含 H3 的标题 DT 互为兄弟，取 dl 前面最近的标题 DT。
func folderOf(a *xhtml.Node) string {
	for el := a.Parent; el != nil; el = el.Parent {
		if el.Type != xhtml.ElementNode {
			continue
		}
		// 情形1：祖先 DT 同时含 H3 与 DL
		if el.Data == "dt" {
			if h3 := firstByTag(el, "h3"); h3 != nil {
				if firstByTag(el, "dl") != nil {
					return nodeText(h3)
				}
			}
		}
		// 情形2：dl 的前面兄弟 DT 含 H3
		if el.Data == "dl" {
			for s := el.PrevSibling; s != nil; s = s.PrevSibling {
				if s.Type == xhtml.ElementNode && s.Data == "dt" {
					if h3 := firstByTag(s, "h3"); h3 != nil {
						return nodeText(h3)
					}
				}
			}
		}
	}
	return ""
}

// noteOf 备注：紧跟书签 DT 的下一个 <DD> 元素（HTML5 解析后的标准形态），
// 其内容为纯文本时即备注；遇到其他结构立即终止。
func noteOf(a *xhtml.Node) string {
	dt := a.Parent
	if dt == nil {
		return ""
	}
	for s := dt.NextSibling; s != nil; s = s.NextSibling {
		if s.Type != xhtml.ElementNode {
			continue
		}
		if s.Data == "dd" {
			structural := false
			for c := s.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == xhtml.ElementNode && (c.Data == "a" || c.Data == "h3" || c.Data == "dl") {
					structural = true
					break
				}
			}
			if !structural {
				return nodeText(s)
			}
			return ""
		}
		if s.Data == "dt" || s.Data == "dl" {
			return ""
		}
	}
	return ""
}
