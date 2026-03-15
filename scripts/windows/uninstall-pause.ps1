#requires -Version 5.1
<#
Usage:
  1) Preview only:
     powershell -ExecutionPolicy Bypass -File .\scripts\windows\uninstall-pause.ps1 -DryRun
  2) Execute cleanup:
     powershell -ExecutionPolicy Bypass -File .\scripts\windows\uninstall-pause.ps1

Notes:
- Removes Pause-related files and registry entries.
- Does not uninstall the system-wide Microsoft Edge WebView2 Runtime.
- Running in an elevated PowerShell is recommended.
#>

param(
  [switch]$DryRun
)

$ErrorActionPreference = "Stop"

function Invoke-DeletePath {
  param([string]$Path)
  try {
    if (Test-Path -LiteralPath $Path -ErrorAction SilentlyContinue) {
      if ($DryRun) {
        Write-Host "[DRYRUN] Remove path: $Path"
      } else {
        Remove-Item -LiteralPath $Path -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "[OK] Removed path: $Path"
      }
    } else {
      Write-Host "[SKIP] Not found: $Path"
    }
  } catch {
    Write-Host "[SKIP] Access denied: $Path"
  }
}

function Invoke-DeleteRegValue {
  param([string]$KeyPath, [string]$ValueName)
  try {
    $item = Get-ItemProperty -Path $KeyPath -ErrorAction Stop
    if ($null -ne $item.$ValueName) {
      if ($DryRun) {
        Write-Host "[DRYRUN] Remove reg value: $KeyPath -> $ValueName"
      } else {
        Remove-ItemProperty -Path $KeyPath -Name $ValueName -ErrorAction SilentlyContinue
        Write-Host "[OK] Removed reg value: $KeyPath -> $ValueName"
      }
    } else {
      Write-Host "[SKIP] Reg value not found: $KeyPath -> $ValueName"
    }
  } catch {
    Write-Host "[SKIP] Reg value not found: $KeyPath -> $ValueName"
  }
}

function Invoke-DeleteRegKey {
  param([string]$KeyPath)
  try {
    if (Test-Path $KeyPath -ErrorAction SilentlyContinue) {
      if ($DryRun) {
        Write-Host "[DRYRUN] Remove reg key: $KeyPath"
      } else {
        Remove-Item -Path $KeyPath -Recurse -Force -ErrorAction SilentlyContinue
        Write-Host "[OK] Removed reg key: $KeyPath"
      }
    } else {
      Write-Host "[SKIP] Reg key not found: $KeyPath"
    }
  } catch {
    Write-Host "[SKIP] Reg key inaccessible: $KeyPath"
  }
}

Write-Host "==== Pause full cleanup start ===="

# 0) Stop running process
Get-Process -Name "Pause" -ErrorAction SilentlyContinue | ForEach-Object {
  if ($DryRun) {
    Write-Host "[DRYRUN] Stop process: $($_.ProcessName) PID=$($_.Id)"
  } else {
    Stop-Process -Id $_.Id -Force -ErrorAction SilentlyContinue
    Write-Host "[OK] Stopped process: $($_.ProcessName) PID=$($_.Id)"
  }
}

# 1) Run bundled uninstaller if present
$uninstaller = "C:\Program Files\Pause\Pause\uninstall.exe"
if (Test-Path $uninstaller -ErrorAction SilentlyContinue) {
  if ($DryRun) {
    Write-Host "[DRYRUN] Run uninstaller: $uninstaller /S"
  } else {
    try {
      Start-Process -FilePath $uninstaller -ArgumentList "/S" -Wait -NoNewWindow
      Write-Host "[OK] Uninstaller executed"
    } catch {
      Write-Host "[WARN] Uninstaller failed; continuing with manual cleanup"
    }
  }
} else {
  Write-Host "[SKIP] Uninstaller not found: $uninstaller"
}

# 2) Remove user-level files for local user profiles (skip system profile aliases)
$excludedProfileNames = @(
  "All Users",
  "Default",
  "Default User",
  "Public",
  "defaultuser0",
  "WDAGUtilityAccount"
)

$profiles = Get-ChildItem "C:\Users" -Directory -Force -ErrorAction SilentlyContinue | Where-Object {
  $_.Name -notin $excludedProfileNames
}

foreach ($p in $profiles) {
  $u = $p.FullName
  Invoke-DeletePath (Join-Path $u "AppData\Roaming\Pause")
  Invoke-DeletePath (Join-Path $u "AppData\Roaming\Pause.exe") # Pause-specific WebView2 user data
  Invoke-DeletePath (Join-Path $u "AppData\Local\Pause")
  Invoke-DeletePath (Join-Path $u ".pause")
  Invoke-DeletePath (Join-Path $u "Desktop\Pause.lnk")
}

# 3) Remove public shortcuts
Invoke-DeletePath "C:\ProgramData\Microsoft\Windows\Start Menu\Programs\Pause.lnk"
Invoke-DeletePath "C:\Users\Public\Desktop\Pause.lnk"

# 4) Remove install directory
Invoke-DeletePath "C:\Program Files\Pause"

# 5) Remove current-user startup entry
Invoke-DeleteRegValue "HKCU:\Software\Microsoft\Windows\CurrentVersion\Run" "com.pause.app"

# 6) Remove startup entry from loaded user hives
$runSuffix = "Software\Microsoft\Windows\CurrentVersion\Run"
Get-ChildItem Registry::HKEY_USERS -ErrorAction SilentlyContinue | ForEach-Object {
  $sid = $_.PSChildName
  if ($sid -match '^S-1-5-21-.+-\d+$') {
    $runKey = "Registry::HKEY_USERS\$sid\$runSuffix"
    Invoke-DeleteRegValue $runKey "com.pause.app"
  }
}

# 7) Remove uninstall registry entries related to Pause
$uninstallRoots = @(
  "HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall",
  "HKLM:\Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall",
  "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall"
)

foreach ($root in $uninstallRoots) {
  if (Test-Path $root -ErrorAction SilentlyContinue) {
    Get-ChildItem $root -ErrorAction SilentlyContinue | ForEach-Object {
      try {
        $k = $_.PSPath
        $v = Get-ItemProperty -Path $k -ErrorAction SilentlyContinue

        $hit = $false
        if ($v.DisplayName -and $v.DisplayName -match 'Pause') { $hit = $true }
        if ($v.InstallLocation -and $v.InstallLocation -match '\\Pause\\') { $hit = $true }
        if ($v.DisplayIcon -and $v.DisplayIcon -match '\\Pause\\') { $hit = $true }
        if ($v.UninstallString -and $v.UninstallString -match '\\Pause\\') { $hit = $true }

        if ($hit) {
          Invoke-DeleteRegKey $k
        }
      } catch {
      }
    }
  }
}

Write-Host "==== Pause full cleanup done ===="
if ($DryRun) {
  Write-Host "This was a dry run. Re-run without -DryRun to actually delete."
}
