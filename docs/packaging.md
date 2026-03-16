# Pause 打包规范（macOS / Windows）

本文档定义 Pause 桌面端的打包规范与发布产物约定，用于统一本地发版与 CI 发版流程。

## 目标

- 统一 macOS 与 Windows 的打包入口和参数语义。
- 固化发布物清单（产物 + 校验和 + 元数据）。
- 保证构建过程可追溯、可复现、可排障。

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

## 发布清单规范

命令：

```bash
./scripts/generate-release-manifest.sh --version <version> --channel <channel>
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

## 推荐发布流程

1. 执行 `./scripts/check-version-sync.sh`，确保版本元信息一致。
2. 执行 macOS 打包（按需传入签名与版本参数）。
3. 执行 Windows 打包（按目标架构分别构建）。
4. 执行 `generate-release-manifest.sh`，生成统一清单与校验文件。
5. 人工验收并归档发布目录。

## GitHub Actions 自动打包

- 工作流文件：`.github/workflows/package-desktop.yml`
- 触发规则：
  - `push` 到 `main`：自动构建 macOS + Windows 并上传 Actions artifacts
  - `push` tag `v*`：在构建基础上自动发布 GitHub Release（附带产物与清单）
  - `workflow_dispatch`：可手动触发打包
- 产出内容：
  - `pause-macos-arm64`：macOS Apple Silicon DMG
  - `pause-macos-x64`：macOS Intel x64 DMG
  - `pause-windows-x64`：Windows 安装包与校验文件
  - `pause-release-manifest`：`release-manifest.txt` + `SHA256SUMS`

## 验收清单

- macOS：DMG 可挂载，`Pause.app` 可拖拽安装，首次启动正常。
- macOS：登录项可启停，Bundle ID 与 helper bundle id 正确。
- Windows：安装、启动、清理流程正常，桌面/开始菜单快捷方式正确。
- Windows：WebView2 策略与目标环境一致（`download`/`browser`/`embed`）。
- 校验：`SHA256SUMS` 与实际上传文件一致。
