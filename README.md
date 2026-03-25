# Pause

Pause 是一个跨平台（macOS / Windows / Linux）的休息提醒应用。当前 macOS 与 Windows 主流程已基本对齐，Linux 提供适配层并持续补齐原生体验。

## 文档索引

- 架构与代码结构：`docs/architecture.md`
- 打包与发布：`docs/packaging.md`

## 项目状态

- 提醒、休息会话、分析、设置、状态栏/托盘与开机启动主流程已接通。
- 当前代码结构与边界说明不再写在 README 中，统一维护在 `docs/architecture.md`。

## 本地开发

### 1) 安装前端依赖

```bash
npm --prefix frontend install
```

### 2) 构建前端资源

```bash
npm --prefix frontend run build
```

### 3) 运行桌面版（Wails）

```bash
go run -tags wails,dev .
```

### 4) 运行无 UI 后端循环

```bash
go run .
```

### 5) 测试

```bash
go test ./...
go test -tags wails ./...
```

## 版本管理

版本号单一来源：`VERSION`

- 校验版本一致性：

```bash
./scripts/check-version-sync.sh
```

- 一次性更新版本（同步 `VERSION` + `wails.json` + `frontend/package.json` + `frontend/package-lock.json`）：

```bash
./scripts/bump-version.sh 0.1.1
```

## 打包与发布（macOS / Windows）

完整规范见：`docs/packaging.md`

### 1) 打包 macOS DMG

```bash
./scripts/build-dmg.sh
```

- 默认产物：
  - `build/bin/macos-arm64/Pause-v<version>-macos-arm64.dmg`
  - `build/bin/macos-x64/Pause-v<version>-macos-x64.dmg`
- 常用参数：
  - `./scripts/build-dmg.sh --version 0.1.0 --no-clean`
  - `./scripts/build-dmg.sh --platform darwin/arm64 --bundle-id com.pause.app --codesign "-" --output-dir build/bin/macos-arm64`
  - `./scripts/build-dmg.sh --platform darwin/amd64 --bundle-id com.pause.app --codesign "-" --output-dir build/bin/macos-x64`
- 查看全部参数：`./scripts/build-dmg.sh --help`

### 2) 打包 Windows 安装器

```bash
./scripts/build-windows-installer.sh
```

- 依赖：本机安装 `NSIS`（macOS 可用 `brew install nsis`）。
- 默认产物目录：`build/bin/windows-x64/`
- 默认安装包文件名：`Pause-v<version>-windows-x64-setup.exe`
- 常用参数：
  - `./scripts/build-windows-installer.sh --platform windows/amd64 --webview2 browser`
  - `./scripts/build-windows-installer.sh --output-dir build/bin/win-release --clean`
- 查看全部参数：`./scripts/build-windows-installer.sh --help`

### 3) 生成统一发布清单与校验和

```bash
./scripts/generate-release-manifest.sh --version 0.1.0 --channel stable
```

- 默认扫描目录：`build/bin`
- 默认输出目录：`build/bin/release`
- 生成文件：
  - `build/bin/release/release-manifest.txt`
  - `build/bin/release/SHA256SUMS`

## 完全清理

macOS：

```bash
./scripts/cleanup/macos/cleanup-pause.sh
```

Windows（PowerShell）：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\cleanup\windows\cleanup-pause.ps1 -DryRun
powershell -ExecutionPolicy Bypass -File .\scripts\cleanup\windows\cleanup-pause.ps1
```

说明：
- 清理脚本已统一放在 `scripts/cleanup/` 下。

## 平台状态

- macOS：主流程可用（托盘、遮罩、通知、声音、开机启动）。
- Windows：主流程可用（空闲检测、托盘菜单、通知、提示音、开机启动、原生遮罩）。
  - 与 macOS 的主要差异：状态栏交互形态不同（Windows 为原生托盘菜单，macOS 为 Popover）。
  - 通知策略：默认使用原生 Win32 通知；若系统通知接口不可用，再回退到 PowerShell 通知兜底。
- Linux：已有适配层，原生体验待补齐。
