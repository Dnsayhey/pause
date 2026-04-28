# Pause 打包、发版与更新源

最后更新：2026-04-02

本文档定义 Pause 当前桌面端的打包规范、GitHub Release 流程以及稳定更新源（stable feed）约定。

## 目标

- 统一 macOS 与 Windows 的打包入口和参数语义。
- 固化发布物清单（产物 + 校验和 + 元数据）。
- 保证构建过程可追溯、可复现、可排障。
- 保证客户端“检查更新”只依赖一个稳定地址。

## 脚本入口

- macOS DMG：`scripts/build-dmg.sh`
- Windows 安装器：`scripts/build-windows-installer.sh`
- 发布清单：`scripts/generate-release-manifest.sh`
- 版本更新：`scripts/bump-version.sh`
- 版本校验：`scripts/check-version-sync.sh`
- 清理脚本目录：`scripts/cleanup/`

## 清理脚本目录规范

- macOS：`scripts/cleanup/macos/cleanup-pause.sh`
- Windows：`scripts/cleanup/windows/cleanup-pause.ps1`

## 版本规范

- 版本单一来源：根目录 `VERSION`
- 发布前先执行 `./scripts/check-version-sync.sh`
- 需要升级版本时仅执行 `./scripts/bump-version.sh <new_version>`，脚本会自动同步：
  - `VERSION`
  - `wails.json` 的 `info.productVersion`
  - `frontend/package.json` 的 `version`
  - `frontend/package-lock.json` 的 `version` 与 `packages[""].version`

## 产物目录规范

- 原始构建产物根目录：`build/bin`
- macOS 产物子目录（默认）：`build/bin/macos-arm64` 与 `build/bin/macos-x64`
- Windows 产物子目录（默认）：`build/bin/windows-x64` 或 `build/bin/windows-arm64`
- 发布清单目录（默认）：`build/bin/release`

建议在正式发版时使用独立目录（例如 `build/bin/release/<version>`）存放归档结果，避免与临时构建文件混放。

## macOS 打包规范

命令：

```bash
./scripts/build-dmg.sh
```

常用参数：

- `--platform <split|darwin/arm64|darwin/amd64|darwin/universal>`：构建目标（默认 `split`，即 arm64+x64 分开构建）
- `--version <version>`：覆盖 `CFBundleShortVersionString/CFBundleVersion`
- `--bundle-id <bundle_id>`：覆盖默认 Bundle ID（默认来自 `internal/meta/bundle_id.txt`）
- `--codesign <identity>`：签名身份（`-` 表示 ad-hoc）
- `--output <path>`：自定义 DMG 输出路径（仅单平台模式）
- `--output-dir <path>`：自定义 DMG 输出目录（单平台）或输出根目录（split 模式）
- `--clean|--no-clean`：是否执行 Wails `-clean`

默认行为：

- 分别产出：
  - `build/bin/macos-arm64/Pause-v<version>-macos-arm64.dmg`
  - `build/bin/macos-x64/Pause-v<version>-macos-x64.dmg`
- 图标来源：`assets/branding/app-icon-1024.png`
- DMG Finder 布局模板：`assets/dmg/dmg-layout.dsstore`（默认把 `Pause.app` 与 `Applications` 固定为左右布局）
- 会自动将登录项 helper 嵌入到 `.app` 并签名。
- 优先使用本机 `wails`，缺失时回退到 `go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`。

示例：

- 分开构建（默认）：
  - `./scripts/build-dmg.sh`
- 只构建 arm64：
  - `./scripts/build-dmg.sh --platform darwin/arm64`
- 只构建 x64：
  - `./scripts/build-dmg.sh --platform darwin/amd64`

## Windows 打包规范

命令：

```bash
./scripts/build-windows-installer.sh
```

前置依赖：

- `NSIS`（`nsis` 或 `makensis` 可执行）

常用参数：

- `--platform <windows/amd64|windows/arm64|...>`
- `--arch-label <label>`
- `--output-dir <path>`
- `--webview2 <download|browser|embed|error>`
- `--clean|--no-clean`

默认行为：

- 默认平台：`windows/amd64`
- 默认目录：`build/bin/windows-x64`
- 默认安装包文件名：`Pause-v<version>-windows-x64-setup.exe`
- 默认从 `scripts/windows-installer/project.nsi` 同步 NSIS 模板到 `build/windows/installer/project.nsi`
- 优先使用本机 `wails`，缺失时回退到 `go run github.com/wailsapp/wails/v2/cmd/wails@v2.10.2`

安装器体验：

- 安装器内置英文和简体中文，英文为默认/兜底语言，简体中文 Windows 会显示中文。
- 首次安装默认路径为 `C:\Program Files\Pause`；用户可在安装过程中自定义目录。
- 安装器会写入 `InstallLocation`，后续升级优先沿用上次安装路径；老版本升级时会从卸载器路径推断旧安装目录。
- 安装或卸载前如果检测到 `Pause.exe` 正在运行，会提示用户先从系统托盘退出 Pause，然后点击“重试”继续。
- 安装完成页默认勾选“Launch Pause / 启动 Pause”，用户可取消勾选。

临时兼容项：

- `0.9.3` 起 Windows 安装器开始写入 `InstallLocation`。
- `UseLegacyInstallDirIfNeeded` 中从 `UninstallString` 推断旧安装目录的逻辑只用于兼容未写入 `InstallLocation` 的老版本。
- 后续已有足够公开版本写入 `InstallLocation`，且不再需要支持这些老版本直接升级时，可以删除该兼容逻辑。

## 发布清单规范

命令：

```bash
./scripts/generate-release-manifest.sh --version <version> --channel stable
```

脚本会扫描以下后缀文件：

- `.dmg`
- `.exe`
- `.msi`
- `.zip`
- `.blockmap`
- `.msix`
- `.appx`

输出：

- `release-manifest.txt`：包含生成时间、版本、渠道、commit、文件条目
- `SHA256SUMS`：发布文件 SHA256 校验和
- `updates.json`：面向客户端消费的机器可读更新元数据

`updates.json` 当前包含：

- `schema_version`
- `generated_at_utc`
- `release.version`
- `release.channel`
- `release.tag`
- `release.commit`
- `release.repository`
- `release.url`
- `assets[]`：每个产物的 `name` / `path` / `os` / `arch` / `kind` / `sha256` / `size` / `url`

建议把 `updates.json` 部署到一个稳定 URL，比如 GitHub Pages 或独立静态站点；客户端只请求这个固定地址，而不是直接依赖 GitHub `latest release` API。

## 当前更新源约定

Pause 当前只维护一个渠道：`stable`。

- 客户端固定请求：`https://dnsayhey.github.io/pause/updates/stable.json`
- `updates.json` 在 Pages 部署时会被复制为：`updates/stable.json`
- 桌面端构建时通过 `VITE_UPDATES_URL` 注入这个固定地址

客户端当前行为：

- 启动后会静默检查一次更新
- 设置页可手动再次检查
- 如果发现更新，会根据当前平台信息匹配最合适的安装包下载地址

平台信息不再由前端 `userAgent` 猜测，而是由 Wails 后端提供 `GetPlatformInfo()`，用于提高 macOS / Windows 包匹配准确性。

当前 GitHub Actions 约定的 Pages 地址格式：

- `stable`：`https://dnsayhey.github.io/pause/updates/stable.json`

前端构建时通过 `VITE_UPDATES_URL` 注入这个稳定地址。

## 推荐发布流程

1. 执行 `./scripts/check-version-sync.sh`，确保版本元信息一致。
2. 执行 macOS 打包（按需传入签名与版本参数）。
3. 执行 Windows 打包（按目标架构分别构建）。
4. 执行 `generate-release-manifest.sh`，生成统一清单与校验文件。
   当前推荐固定使用 `stable` 渠道标签，与自动更新地址保持一致。
5. 人工验收并归档发布目录。

## GitHub Actions 自动打包

- 工作流文件：`.github/workflows/package-desktop.yml`
- 触发规则：
  - `push` tag `v*`：自动构建并发布 GitHub Release（附带产物与清单）
  - `workflow_dispatch`：可在 GitHub Actions 页面手动触发构建
- 产出内容：
  - `pause-macos-arm64`：macOS Apple Silicon DMG
  - `pause-macos-x64`：macOS Intel x64 DMG
  - `pause-windows-x64`：Windows 安装包与校验文件
  - `pause-release-manifest`：`release-manifest.txt` + `SHA256SUMS` + `updates.json`
- Pages：
  - tag 发版成功后自动部署 `updates/stable.json`
  - 同时生成根页 `index.html`，方便人工检查 Pages 是否正常可访问

## 推荐发布动作

### 本地准备

1. 执行 `./scripts/bump-version.sh <new_version>`
2. 执行 `./scripts/check-version-sync.sh`
3. 运行必要测试与构建校验
4. 提交代码并创建 tag：`v<new_version>`

### 远端发布

1. push `main`
2. push tag `v<new_version>`
3. 等待 GitHub Actions 完成：
   - 桌面安装包构建
   - Release 发布
   - `stable.json` 更新
   - GitHub Pages 部署

## 验收清单

- macOS：DMG 可挂载，`Pause.app` 可拖拽安装，首次启动正常。
- macOS：登录项可启停，Bundle ID 与 helper bundle id 正确。
- Windows：安装、启动、清理流程正常，桌面/开始菜单快捷方式正确。
- Windows：WebView2 策略与目标环境一致（`download`/`browser`/`embed`）。
- 校验：`SHA256SUMS` 与实际上传文件一致。
- 更新：`https://dnsayhey.github.io/pause/updates/stable.json` 可访问且版本号正确。
