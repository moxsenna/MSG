@echo off
setlocal
echo === MSG Desktop Build FAST (skip bindings/frontend) ===
echo Gunakan ini SETELAH build pertama sukses sekali.
echo Build pertama tetap butuh 5-15 menit, jangan di-Ctrl+C.
echo.
wails build -s -skipbindings -m -o MSGDesktop.exe
if errorlevel 1 (
  echo GAGAL fast build
  pause
  exit /b 1
)
echo.
echo SELESAI - build\bin\MSGDesktop.exe
dir build\bin\MSGDesktop.exe
pause
