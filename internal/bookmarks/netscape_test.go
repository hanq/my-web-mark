package bookmarks

import (
	"strings"
	"testing"
)

func sampleItems() []ExportBookmark {
	return []ExportBookmark{
		{Title: "Vue", URL: "https://vuejs.org", Note: "框架文档", Folder: "前端", Tags: []string{"前端", "框架"}},
		{Title: "Go", URL: "https://go.dev", Folder: "前端", Tags: []string{"前端"}},
		{Title: "无标签站", URL: "https://example.com", Note: "示例", Pinned: true},
	}
}

func TestNetscapeRoundTrip(t *testing.T) {
	items := sampleItems()
	htmlOut := RenderNetscapeHTML("开发书签", items)

	name, got, err := ParseNetscapeHTML(htmlOut)
	if err != nil {
		t.Fatal(err)
	}
	if name != "开发书签" {
		t.Fatalf("主题名错误: %q", name)
	}
	if len(got) != len(items) {
		t.Fatalf("条数不一致: %d vs %d (%+v)", len(got), len(items), got)
	}
	byURL := map[string]ExportBookmark{}
	for _, it := range got {
		byURL[it.URL] = it
	}
	if v := byURL["https://vuejs.org"]; v.Title != "Vue" || v.Note != "框架文档" || v.Folder != "前端" {
		t.Fatalf("Vue 条目错误: %+v", v)
	}
	if v := byURL["https://example.com"]; v.Folder != "" || v.Note != "示例" {
		t.Fatalf("无标签条目错误: %+v", v)
	}
}

func TestJSONRoundTrip(t *testing.T) {
	items := sampleItems()
	data, err := MarshalThemeFile(ExportThemeMeta{Name: "开发", Color: "#4f8cff"}, items)
	if err != nil {
		t.Fatal(err)
	}
	if DetectExportFormat(string(data)) != "json" {
		t.Fatal("JSON 格式检测失败")
	}
	f, err := ParseThemeFile(data)
	if err != nil {
		t.Fatal(err)
	}
	if f.Theme.Name != "开发" || f.Theme.Color != "#4f8cff" || len(f.Bookmarks) != 3 {
		t.Fatalf("JSON 解析不一致: %+v", f)
	}
	if len(f.Bookmarks[0].Tags) != 2 || !f.Bookmarks[2].Pinned {
		t.Fatalf("标签/置顶丢失: %+v", f.Bookmarks[0])
	}
}

func TestDetectFormatHTML(t *testing.T) {
	out := RenderNetscapeHTML("x", sampleItems())
	if DetectExportFormat(out) != "html" {
		t.Fatal("HTML 格式检测失败")
	}
}

func TestParseBrowserStyleHTML(t *testing.T) {
	// 浏览器导出的典型结构（含 DD 备注、嵌套文件夹）
	doc := `<!DOCTYPE NETSCAPE-Bookmark-file-1>
<META HTTP-EQUIV="Content-Type" CONTENT="text/html; charset=UTF-8">
<TITLE>Bookmarks</TITLE>
<H1>书签栏</H1>
<DL><p>
    <DT><H3>工具</H3>
    <DL><p>
        <DT><A HREF="https://a.com">站点A</A>
        <DD>备注A
    </DL><p>
    <DT><A HREF="https://b.com">站点B</A>
</DL><p>`
	name, items, err := ParseNetscapeHTML(doc)
	if err != nil {
		t.Fatal(err)
	}
	if name != "书签栏" {
		t.Fatalf("主题名: %q", name)
	}
	if len(items) != 2 {
		t.Fatalf("条数: %d", len(items))
	}
	if items[0].Folder != "工具" || items[0].Note != "备注A" {
		t.Fatalf("文件夹/备注解析错误: %+v", items[0])
	}
	if items[1].Folder != "" {
		t.Fatalf("顶层书签不应有文件夹: %+v", items[1])
	}
	if !strings.Contains(RenderNetscapeHTML("t", items), "站点A") {
		t.Fatal("渲染缺失")
	}
}
