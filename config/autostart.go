package config

import "golang.org/x/sys/windows/registry"

const (
	regRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`
	regName   = "ClaudeTrafficLight"
)

// AutostartEnabled 查询注册表：指定 exe 是否已设为开机自启。
func AutostartEnabled(exePath string) bool {
	val, err := AutostartGet()
	if err != nil {
		return false
	}
	return val == exePath
}

// AutostartGet 读注册表开机自启值。返回 exe 路径或 error。
func AutostartGet() (string, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, regRunKey, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	s, _, err := k.GetStringValue(regName)
	return s, err
}

// AutostartSet 写注册表开机自启（指向指定 exe 路径）。
func AutostartSet(exePath string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(regName, exePath)
}

// AutostartRemove 删注册表开机自启记录。
func AutostartRemove() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, regRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue(regName)
}

// SyncAutostart 路径自校正：按 cfg.Startup 意图对齐注册表（exe 换盘则更新、残留则清理）。
// 在 main 启动时调用。
func SyncAutostart(exePath string, cfgStartup bool) {
	regVal, regErr := AutostartGet()
	regExists := regErr == nil

	if cfgStartup {
		if regExists && regVal == exePath {
			return
		}
		_ = AutostartSet(exePath)
	} else {
		if !regExists {
			return
		}
		_ = AutostartRemove()
	}
}

// ToggleAutostart 在菜单开关切换时调用：enable → 写注册表，disable → 删。
func ToggleAutostart(exePath string, enable bool) error {
	if enable {
		return AutostartSet(exePath)
	}
	return AutostartRemove()
}
