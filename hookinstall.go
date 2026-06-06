package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// 4 个生命周期 hook → 状态。PreToolUse/PostToolUse 需要 matcher，其余不需要。
var hookEvents = []struct {
	event   string
	state   string
	matcher bool
}{
	{"UserPromptSubmit", "thinking", false},
	{"PreToolUse", "running", true},
	{"PostToolUse", "thinking", true},
	{"Stop", "idle", false},
}

// installHooks 把挂件的 4 个状态 hook 安全合并进 ~/.claude/settings.json。
// 原则：settings.json 必须已存在（即本机已装 Claude Code）才合并；不存在则
// 静默跳过——不在没装 Claude Code 的机器上创建任何目录或文件。
// 已有文件时：幂等（已存在则只更新自己那条的路径，绝不重复加）、只增不删
// （别人的配置一字不动）、先备份（写前存 settings.json.bak）。
func installHooks() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	exe, err := os.Executable()
	if err != nil {
		return
	}
	settingsPath := filepath.Join(home, ".claude", "settings.json")

	// 读现有 settings。文件不存在 = 本机没装 Claude Code → 静默跳过，不创建。
	orig, err := os.ReadFile(settingsPath)
	if err != nil {
		return
	}
	var root map[string]interface{}
	if json.Unmarshal(orig, &root) != nil {
		return
	}

	hooks, _ := root["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
	}

	changed := false
	for _, he := range hookEvents {
		if mergeHookEvent(hooks, he.event, he.state, he.matcher, exe) {
			changed = true
		}
	}
	if !changed {
		return // 已是最新，不写、不备份
	}
	root["hooks"] = hooks

	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return
	}
	// 先备份原文件，再写回（能走到这里说明原文件已存在且读取成功）
	os.WriteFile(settingsPath+".bak", orig, 0644)
	os.WriteFile(settingsPath, out, 0644)
}

// mergeHookEvent 在 hooks[event] 里插入/更新挂件自己的 hook 组，返回是否有改动。
// 靠 command 的 basename 识别「我加的」：已存在且路径相同→不动；路径变了→更新；
// 不存在→追加。别人的 hook 组完全不碰。
func mergeHookEvent(hooks map[string]interface{}, event, state string, matcher bool, exe string) bool {
	ourCmd := map[string]interface{}{
		"type":    "command",
		"command": exe,
		"args":    []interface{}{"hook", state},
	}
	ourGroup := map[string]interface{}{
		"hooks": []interface{}{ourCmd},
	}
	if matcher {
		ourGroup["matcher"] = "*"
	}

	arr, _ := hooks[event].([]interface{})
	for i, g := range arr {
		grp, ok := g.(map[string]interface{})
		if !ok {
			continue
		}
		inner, _ := grp["hooks"].([]interface{})
		for _, h := range inner {
			hm, ok := h.(map[string]interface{})
			if !ok {
				continue
			}
			cmd, _ := hm["command"].(string)
			if strings.EqualFold(filepath.Base(cmd), filepath.Base(exe)) {
				if cmd == exe {
					return false // 已存在且路径未变，无需改动
				}
				arr[i] = ourGroup // 路径变了（如换 U 盘盘符），更新自己这条
				hooks[event] = arr
				return true
			}
		}
	}
	hooks[event] = append(arr, ourGroup) // 没有则追加
	return true
}
