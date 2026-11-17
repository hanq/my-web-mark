package app

import (
	"fmt"
)

// CopyTextInput 剪贴板写入入参。
type CopyTextInput struct {
	Text string `json:"text"`
}

// CopyToClipboard 将文本写入系统剪贴板（复制账号密码用）。
// 剪贴板自动清空策略由前端计时调用 ClearClipboard 实现。
func (b *BrowserService) CopyToClipboard(in CopyTextInput) error {
	if in.Text == "" {
		return fmt.Errorf("app: 内容为空")
	}
	return copyToClipboard(in.Text)
}

// ClearClipboard 清空剪贴板（敏感信息复制后定时调用）。
func (b *BrowserService) ClearClipboard() error {
	return copyToClipboard("")
}
