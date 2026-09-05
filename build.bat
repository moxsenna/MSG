@echo off
setlocal
echo === MSG Desktop Build 1-Klik (Production) ===
echo.

REM 1. Build frontend (Vite)
echo [1/3] Building frontend...
pushd desktop\frontend
call npm run build
if errorlevel 1 (
  echo GAGAL build frontend
  pause
  exit /b 1
)
popd
echo Frontend OK -> desktop\frontend\dist

REM 2. Vet & Test
echo.
echo [2/3] Verifikasi Go...
go vet ./internal/msg/... ./desktop/... ./cmd/msgdesktop/...
if errorlevel 1 (
  echo go vet GAGAL
  pause
  exit /b 1
)
go test ./internal/msg/... ./desktop/... -count=1
if errorlevel 1 (
  echo go test GAGAL
  pause
  exit /b 1
)

REM 3. Build exe Wails (wajib pakai wails build, bukan go build)
echo.
echo [3/3] Building exe Wails (bisa 5-15 menit pertama kali)...
if not exist build\bin mkdir build\bin
wails build -clean -s -m -o MSGDesktop.exe
if errorlevel 1 (
  echo wails build GAGAL - coba wails build -s -m -v 2 untuk log detail
  pause
  exit /b 1
)

echo.
echo === SELESAI ===
echo File: build\bin\MSGDesktop.exe
dir build\bin\MSGDesktop.exe
echo.
echo Jalankan: build\bin\MSGDesktop.exe
echo Data:    %%LOCALAPPDATA%%\MSG\msg.db
echo.
pause
