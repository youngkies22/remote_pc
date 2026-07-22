# Skrip build Remote PC: menghasilkan biner server & agent untuk amd64 dan 386.
# Jalankan dari root project:  powershell -ExecutionPolicy Bypass -File build.ps1
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root
New-Item -ItemType Directory -Force -Path bin | Out-Null

$archs = @("amd64", "386")
# -H=windowsgui: kompilasi sebagai aplikasi GUI-subsystem sehingga Windows TIDAK
# pernah membuat jendela console (terminal hitam) saat exe dijalankan. Semua
# umpan balik ke user memakai dialog (MessageBox); log tetap ke file.
$ldflags = "-s -w -H=windowsgui"

# Info build ditanam ke internal/version supaya bisa dicek dari halaman
# /version — berguna memastikan rebuild benar-benar mengambil kode terbaru.
# AppVersion = "v" + jumlah commit git: nomor urut yang naik otomatis tiap ada
# perubahan, tanpa perlu di-bump manual (biar selalu akurat & tak pernah lupa).
$commitCount = (git rev-list --count HEAD 2>$null)
$appVersion = if ($commitCount) { "v$commitCount" } else { "dev" }
$gitCommit = (git rev-parse --short HEAD 2>$null)
if (-not $gitCommit) { $gitCommit = "unknown" }
$buildTime = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
$serverLdflags = "$ldflags -X remote_pc/internal/version.AppVersion=$appVersion -X remote_pc/internal/version.GitCommit=$gitCommit -X remote_pc/internal/version.BuildTime=$buildTime"

foreach ($arch in $archs) {
    $env:GOOS = "windows"
    $env:GOARCH = $arch
    $env:CGO_ENABLED = "0"

    Write-Host "==> build server ($arch)"
    go build -ldflags $serverLdflags -o "bin/server-$arch.exe" ./cmd/server
    if ($LASTEXITCODE -ne 0) { throw "build server $arch gagal" }

    Write-Host "==> build agent ($arch)"
    go build -ldflags $ldflags -o "bin/agent-$arch.exe" ./cmd/agent
    if ($LASTEXITCODE -ne 0) { throw "build agent $arch gagal" }
}

Write-Host ""
Write-Host "Selesai. Biner ada di folder bin/:" -ForegroundColor Green
Get-ChildItem bin | Format-Table Name, Length
