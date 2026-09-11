$ErrorActionPreference = "Continue"
$env:GOTOOLCHAIN = "local"
$env:GOPATH = "$env:USERPROFILE\go"
$env:PATH = "$env:LOCALAPPDATA\go\bin;${env:USERPROFILE}\go\bin;$env:PATH"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

# Внешние команды запускаем с перенаправлением stderr и проверкой $LASTEXITCODE,
# иначе любой warning в stderr ложно прерывает сборку. Сбой команды всё же ловится.
function Invoke-Checked {
    param([scriptblock]$Command, [string]$What)
    & $Command 2>$null
    if ($LASTEXITCODE -ne 0) { throw "$What failed (exit $LASTEXITCODE)" }
}

Write-Host "==> Frontend: npm install + build"
Push-Location "frontend"
Invoke-Checked { npm install --no-fund --no-audit } "npm install"
Invoke-Checked { npm run build } "npm build"
Pop-Location

Write-Host "==> Resources: winres (icon + manifest)"
Invoke-Checked { go run github.com/tc-hib/go-winres@v0.3.1 make --in resources\winres.json --out rsrc_windows_amd64.syso --arch amd64 } "winres"

Write-Host "==> Launcher: go build (onefile, windows gui, stripped)"
New-Item -ItemType Directory -Force -Path "dist" | Out-Null
Invoke-Checked { go build -ldflags "-H=windowsgui -s -w" -o "dist\dzglauncher.exe" . } "go build"

Copy-Item -Force "config.json" "dist\config.json"

$exe = Get-Item "dist\dzglauncher.exe"
Write-Host ""
Write-Host "==> Done:"
Write-Host "    $($exe.FullName) ($([math]::Round($exe.Length / 1MB, 1)) MB)"
Write-Host "    config.json next to exe overrides embedded defaults."