# Pause

Pause 是一个跨平台休息提醒应用（macOS / Windows / Linux）。

## 文档索引

- [架构与代码结构](./docs/architecture.md)
- [通知能力与前端策略](./docs/notification-logic.md)
- [打包与发布](./docs/packaging.md)

## 开发环境

- Go `1.24+`
- Node.js + npm（用于 frontend 构建）

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

### 5) 运行测试

```bash
go test ./...
go test -tags wails ./...
```

## 版本管理

版本号单一来源：`VERSION`

```bash
# 校验版本一致性
./scripts/check-version-sync.sh

# 更新版本（同步 VERSION / wails.json / frontend package.json）
./scripts/bump-version.sh <new_version>
```

## 打包与发布

完整规范见：`docs/packaging.md`

```bash
# macOS DMG
./scripts/build-dmg.sh

# Windows 安装器
./scripts/build-windows-installer.sh

# 生成发布清单与校验和
./scripts/generate-release-manifest.sh --version <version> --channel <channel>
```

## 清理脚本

```bash
# macOS
./scripts/cleanup/macos/cleanup-pause.sh
```

```powershell
# Windows
powershell -ExecutionPolicy Bypass -File .\scripts\cleanup\windows\cleanup-pause.ps1 -DryRun
powershell -ExecutionPolicy Bypass -File .\scripts\cleanup\windows\cleanup-pause.ps1
```

## 平台说明

- macOS / Windows：主流程可用（提醒、休息会话、通知、开机启动、桌面壳交互）。
- Linux：提供适配层，桌面体验持续补齐。
