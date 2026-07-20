# Skrip build AGENT khusus Windows 7 (client saja).
#
# Kenapa terpisah dari build.ps1 utama:
#   Sejak Go 1.21, Google MENGHAPUS dukungan Windows 7/8/Server 2012 — biner
#   yang dibuat Go >= 1.21 langsung crash saat start di Win7. Go 1.20.14 adalah
#   versi Go TERAKHIR yang mendukung Win7. Modul di folder win7/ ini memakai
#   go.mod sendiri (go 1.20 + dependency versi lama yang kompatibel) sehingga
#   proyek utama (server + agent modern) tetap di Go 1.25 tanpa terpengaruh.
#
# GOTOOLCHAIN=go1.20.14 membuat perintah `go` otomatis mengunduh & memakai
# toolchain 1.20.14 walau Go yang terpasang di mesin ini versi baru.
#
# Jalankan dari folder win7/:  powershell -ExecutionPolicy Bypass -File build-win7.ps1
$ErrorActionPreference = "Stop"

$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root
New-Item -ItemType Directory -Force -Path ../bin | Out-Null

$env:GOTOOLCHAIN = "go1.20.14"   # versi Go terakhir yang mendukung Windows 7
$env:GOOS = "windows"
$env:CGO_ENABLED = "0"
# -H=windowsgui: tanpa jendela console (terminal hitam); umpan balik lewat MessageBox.
$ldflags = "-s -w -H=windowsgui"

# 386 = 32-bit: SATU exe ini jalan di Windows 7 32-bit MAUPUN 64-bit.
# amd64 = opsional, khusus Win7 64-bit (sedikit lebih optimal, tapi 386 sudah cukup).
$archs = @("386", "amd64")

foreach ($arch in $archs) {
    $env:GOARCH = $arch
    Write-Host "==> build agent-win7 ($arch) dengan Go 1.20.14"
    go build -ldflags $ldflags -o "../bin/agent-win7-$arch.exe" ./cmd/agent
    if ($LASTEXITCODE -ne 0) { throw "build agent-win7 $arch gagal" }
}

Write-Host ""
Write-Host "Selesai. Biner Win7 ada di folder bin/:" -ForegroundColor Green
Get-ChildItem ../bin/agent-win7-*.exe | Format-Table Name, Length
