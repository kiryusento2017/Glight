# CHANGELOG

## v1.5.1（2026-06-07）

**修复**

- 修复进程检测误判 Claude Desktop 为 Claude Code（`watcher/procmon.go` 改用 `QueryFullProcessImageNameW` 取完整路径，要求以 `\.local\bin\claude.exe` 结尾才认定为 Claude Code；此前仅靠进程名 `claude.exe` 匹配，Claude Desktop 同名进程也在运行时会误判「Claude Code 在线」，导致挂件无法降灰）

**新增**

- 新增 `--demo` 命令行开关（解除 `WDA_EXCLUDEFROMCAPTURE` 捕获排除，供 OBS / 录屏软件采集挂件窗口画面；正常使用不受影响，重启恢复排除）
- 新增 MIT LICENSE 文件（明确开源协议）
- 新增日/韩/繁中三语 README（`docs/readme/README.ja.md`、`README.ko.md`、`README.zh-TW.md`，语言切换扩展至 5 种：中/英/日/韩/繁中）

**品牌**

- 品牌全面统一为 Glight（exe 属性 `ProductName` / `FileDescription`、README 标题、CLAUDE.md 全部同步）
- 发行版文件名改为 `Glight-v<版本号>-windows-amd64.exe`（遵循 Go 项目发行包标准命名规范）
- `versioninfo.json` 版本号升至 1.5.1，`rsrc_windows_amd64.syso` 同步重生成

**文档**

- 重写 README（中英双语主文档重构，补全状态探测机制/多 agent 并发/hook 写入文件/注册表自启等说明）
- 安装说明新增「会动哪些文件」明细（`~/.claude/settings.json` hook 合并、`~/.claude/agent-light/` 状态文件、`HKCU\Run` 注册表键）
- CLAUDE.md 写入发行六步强制流程（① 问版本号 → ② 改 `versioninfo.json` → ③ 重生成 syso → ④ 构建 → ⑤ 验证版本信息 → ⑥ 推 tag；用户在 GitHub 网页上传 exe 建 Release）
- CLAUDE.md 修正三处过时内容（`procmon.go` 模块描述、当前进度日期、已删 `_liquid-glass-ref/` 引用）
- `docs/编译构建发行.md` 补注「发行前必须先确认版本号」规范

**整理**

- 删除未使用的 `app.manifest`（早期遗留，DPI 感知改由代码 `SetProcessDpiAwarenessContext` 注册，无需 manifest）
- README 翻译版归档至 `docs/readme/`（原根目录 `README.en.md` 移入，首页语言切换链接同步更新）
- `docs/superpowers/specs/` 重命名为 `docs/specs/`

---

## v1.5.0（2026-06-06）

**新增**

- 新增编译构建发行流程文档 `docs/编译构建发行.md`（调试 `go run` / 编译 exe 唯一命令 / 发行三场景 + 命令速查）
- 新增 exe 版本信息嵌入（goversioninfo + `versioninfo.json`，产品名 Claude Code Light / 署名「终末诗篇」）
- 新增统一编译铁律（调试用 `go run`，凡是产出 exe 只有一条命令、四件防护一次带齐，杜绝误发带本机特征或带黑窗的版本）
- 新增 `dist/` 输出隔离（发行版 exe 统一输出到 `dist/` 文件夹，与源码隔离，已加入 `.gitignore`）
- 新增 README 构建章节编译铁律说明 + 图标段更新为 goversioninfo 方案
- 新增发布防报毒 SOP `docs/发布清单.md`（VirusTotal 自检 / 误报申诉入口 / 为什么不走代码签名）

**更新**

- 更新软件图标为多尺寸带透明通道的 ico（256/64/48/32/16px）
- 更新 CLAUDE.md 构建命令/规则/syso 章节（旧 `rsrc` 方案替换为 goversioninfo 流程）
