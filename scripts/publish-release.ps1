param([Parameter(Mandatory = $true)][string]$Version, [string]$LocalDir = "")
$ErrorActionPreference = "Stop"
if ($Version -notmatch '^v\d+\.\d+\.\d+$') { throw "Version must look like v0.2.0" }
$root = "D:\Scrape Data\google-maps-scraper-gosom"
Set-Location $root
Write-Host "=== Build frontend ==="
Set-Location desktop\frontend; npm run build
if ($LASTEXITCODE -ne 0) { throw "frontend build failed" }
Set-Location $root
Write-Host "=== Vet ==="
go vet ./cmd/msgdesktop/... ./internal/msg/... ./desktop/...
if ($LASTEXITCODE -ne 0) { throw "vet failed" }
Write-Host "=== Refresh icon+version resources ==="
$ver = $Version.TrimStart("v")
$wj = Get-Content "build\windows\winres.json" -Raw
$wj = $wj -replace '"file_version": "[^"]*"', "`"file_version`": `"$ver.0`""
$wj = $wj -replace '"product_version": "[^"]*"', "`"product_version`": `"$ver.0`""
$wj = $wj -replace '"FileVersion": "[^"]*"', "`"FileVersion`": `"$ver`""
$wj = $wj -replace '"ProductVersion": "[^"]*"', "`"ProductVersion`": `"$ver`""
Set-Content "build\windows\winres.json" -Value $wj -NoNewline
Set-Location build\windows
go run github.com/tc-hib/go-winres@latest make --arch amd64 --in winres.json --out ..\..\cmd\msgdesktop\rsrc_windows_amd64
if ($LASTEXITCODE -ne 0) { throw "go-winres failed" }
Move-Item "$root\cmd\msgdesktop\rsrc_windows_amd64_windows_amd64.syso" "$root\cmd\msgdesktop\rsrc_windows_amd64.syso" -Force -ErrorAction SilentlyContinue
Set-Location $root
Write-Host "=== Build exe $Version (per-user, no admin needed for self-update) ==="
New-Item -ItemType Directory -Force -Path "build\out" | Out-Null
go build -buildvcs=false -tags desktop,production -ldflags "-w -s -H windowsgui -X main.appVersion=$Version" -o "build\out\MSGDesktop.exe" ./cmd/msgdesktop
if ($LASTEXITCODE -ne 0) { throw "go build failed" }
$env:Path += ";C:\Program Files (x86)\NSIS"
Set-Location build\windows\installer
& "C:\Program Files (x86)\NSIS\makensis.exe" -DWAILS_INSTALL_SCOPE=user -DREQUEST_EXECUTION_LEVEL=user -DARG_WAILS_AMD64_BINARY="..\..\out\MSGDesktop.exe" project.nsi
if ($LASTEXITCODE -ne 0) { throw "makensis failed" }
Set-Location $root
Write-Host "=== Checksums ==="
$exe = "build\out\MSGDesktop.exe"
$hash = (Get-FileHash $exe -Algorithm SHA256).Hash.ToLower()
"$hash  MSGDesktop.exe" | Set-Content "build\out\MSGDesktop.exe.sha256" -NoNewline
$inst = "build\out\MSG Desktop-amd64-installer.exe"
$instHash = (Get-FileHash $inst -Algorithm SHA256).Hash.ToLower()
"$instHash  MSG.Desktop-amd64-installer.exe" | Set-Content "build\out\MSG.Desktop-amd64-installer.exe.sha256" -NoNewline
Write-Host "sha256 exe: $hash"
Write-Host "sha256 installer: $instHash"
Write-Host "=== Publish to moxsenna/msg-desktop ==="
gh release create $Version "$exe" "build\out\MSGDesktop.exe.sha256" "$inst" "build\out\MSG.Desktop-amd64-installer.exe.sha256" --repo moxsenna/msg-desktop --title "MSG Desktop $Version" --notes "Auto-update release. Install fresh via installer, selanjutnya update dari dalam aplikasi (Settings - Update)."
if ($LocalDir -ne "") {
  Write-Host "=== Copy to local dir $LocalDir (neutral names dodge AV filename blocks) ==="
  New-Item -ItemType Directory -Force -Path $LocalDir | Out-Null
  Copy-Item $exe (Join-Path $LocalDir "msg-app.bin") -Force
  Copy-Item "build\out\MSGDesktop.exe.sha256" (Join-Path $LocalDir "msg-app.bin.sha256") -Force
  Copy-Item $inst (Join-Path $LocalDir "msg-setup.bin") -Force
  @{ version = $Version; notes = "Rilis lokal $Version (tanpa GitHub)."; exe = "msg-app.bin"; installer = "msg-setup.bin" } | ConvertTo-Json | Set-Content (Join-Path $LocalDir "latest.json")
  Remove-Item (Join-Path $LocalDir "MSGDesktop.exe") -Force -ErrorAction SilentlyContinue
  Remove-Item (Join-Path $LocalDir "MSGDesktop.exe.sha256") -Force -ErrorAction SilentlyContinue
  Remove-Item (Join-Path $LocalDir "MSG.Desktop-amd64-installer.exe") -Force -ErrorAction SilentlyContinue
  Write-Host "local dir ready"
}
Write-Host "DONE $Version"
