$ErrorActionPreference = "Stop"
$env:GOTOOLCHAIN = "local"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

Write-Host "==> Frontend: npm install + build"
Push-Location "frontend"
npm install
if ($LASTEXITCODE -ne 0) { throw "npm install failed" }
npm run build
if ($LASTEXITCODE -ne 0) { throw "npm build failed" }
Pop-Location

Write-Host "==> Resources: winres (icon + manifest)"
go run github.com/tc-hib/go-winres@v0.3.1 make --in resources\winres.json --out rsrc_windows_amd64.syso --arch amd64
if ($LASTEXITCODE -ne 0) { throw "winres failed" }

Write-Host "==> Launcher: go build (onefile, windows gui, stripped)"
New-Item -ItemType Directory -Force -Path "dist" | Out-Null
go build -ldflags "-H=windowsgui -s -w" -o "dist\dzglauncher.exe" .
if ($LASTEXITCODE -ne 0) { throw "go build failed" }

Copy-Item -Force "config.json" "dist\config.json"

$exe = Get-Item "dist\dzglauncher.exe"
Write-Host ""
Write-Host "==> Done:"
Write-Host "    $($exe.FullName) ($([math]::Round($exe.Length / 1MB, 1)) MB)"
Write-Host "    config.json next to exe overrides embedded defaults."