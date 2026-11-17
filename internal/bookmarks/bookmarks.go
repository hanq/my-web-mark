// Package bookmarks 负责各浏览器（Edge/Chrome/Zen/Firefox）本地书签的
// 定位、解析与运行状态检测。
package bookmarks

import (
	"fmt"
	"time"
)

// BrowserKind 浏览器类型标识。
type BrowserKind string

const (
	BrowserEdge    BrowserKind = "edge"
	BrowserChrome  BrowserKind = "chrome"
	BrowserZen     BrowserKind = "zen"
	BrowserFirefox BrowserKind = "firefox"
)

// AllBrowsers 支持检测的全部浏览器。
var AllBrowsers = []BrowserKind{BrowserEdge, BrowserChrome, BrowserZen, BrowserFirefox}

// Source 名称与 BrowserKind 一致（bookmark.source 字段取值）。
func (k BrowserKind) String() string { return string(k) }

// Node 书签树节点（文件夹或书签）。
type Node struct {
	Name     string  `json:"name"`               // 文件夹名或书签标题
	URL      string  `json:"url,omitempty"`      // 书签地址；文件夹为空
	Children []*Node `json:"children,omitempty"` // 子节点
}

// IsFolder 是否为文件夹（无 URL 即文件夹）。
func (n *Node) IsFolder() bool { return n.URL == "" }

// Flat 扁平书签条目（解析结果展开后）。
type Flat struct {
	Title    string    `json:"title"`
	URL      string    `json:"url"`
	Folder   string    `json:"folder"`   // 所属文件夹路径，如 "书签栏/技术文档"
	AddedAt  time.Time `json:"addedAt"`  // 添加时间（可能为零值）
}

// flattenFolderPath 递归展开节点树，folder 为当前路径前缀。
func flatten(nodes []*Node, folder string, out *[]Flat) {
	for _, n := range nodes {
		if n == nil {
			continue
		}
		if n.IsFolder() {
			sub := folder
			if sub != "" {
				sub += "/"
			}
			sub += n.Name
			flatten(n.Children, sub, out)
		} else {
			*out = append(*out, Flat{Title: n.Name, URL: n.URL, Folder: folder})
		}
	}
}

// Flatten 将书签树展开为扁平列表（跳过空文件夹）。
func Flatten(roots ...*Node) []Flat {
	var out []Flat
	flatten(roots, "", &out)
	seen := make(map[string]bool, len(out))
	res := out[:0]
	for _, f := range out {
		if f.URL == "" || seen[f.URL] {
			continue // 去 URL 空项与重复
		}
		seen[f.URL] = true
		res = append(res, f)
	}
	return res
}

// Status 单个浏览器的检测状态。
type Status struct {
	Browser   BrowserKind `json:"browser"`
	Name      string      `json:"name"`      // 显示名
	Running   bool        `json:"running"`   // 进程是否运行中
	Installed bool        `json:"installed"` // 是否找到数据目录
	Bookmarks int         `json:"bookmarks"` // 可导入书签数量（-1 表示读取失败）
	Error     string      `json:"error,omitempty"`
}

// DisplayNames 浏览器显示名。
var DisplayNames = map[BrowserKind]string{
	BrowserEdge:    "Microsoft Edge",
	BrowserChrome:  "Google Chrome",
	BrowserZen:     "Zen Browser",
	BrowserFirefox: "Mozilla Firefox",
}

// ErrNoProfile 未找到浏览器 Profile 时返回。
var ErrNoProfile = fmt.Errorf("bookmarks: 未找到浏览器数据目录")
