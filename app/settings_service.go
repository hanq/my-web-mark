package app

import (
	"fmt"
)

// 设置键常量（settings 表 KV）。
const (
	// SettingKeyLaunchBrowser 书签打开方式：system/edge/chrome/zen/firefox
	SettingKeyLaunchBrowser = "launch_browser"
)

// launchBrowserValues launch_browser 允许的取值。
var launchBrowserValues = map[string]bool{
	"system": true, "edge": true, "chrome": true, "zen": true, "firefox": true,
}

// SettingsService 应用设置服务（KV，前端可直接读写白名单键）。
type SettingsService struct {
	sh *shared
}

// settingSchema 键 -> 允许值集合（空 set 表示无值约束）。
var settingSchema = map[string]map[string]bool{
	SettingKeyLaunchBrowser: launchBrowserValues,
}

// Get 读取设置值；未设置返回空串。
func (s *SettingsService) Get(key string) (string, error) {
	if _, known := settingSchema[key]; !known {
		return "", fmt.Errorf("app: 未知的设置项 %q", key)
	}
	return s.sh.store.GetSetting(key)
}

// Set 保存设置值；键与值均需在白名单内。
func (s *SettingsService) Set(key, value string) error {
	allowed, known := settingSchema[key]
	if !known {
		return fmt.Errorf("app: 未知的设置项 %q", key)
	}
	if len(allowed) > 0 && !allowed[value] {
		return fmt.Errorf("app: 设置项 %q 的值非法", key)
	}
	return s.sh.store.SetSetting(key, value)
}
