@echo off
setlocal
cd /d "%~dp0"

echo ========================================
echo   TDY Manager - Build and Start
echo ========================================
echo.

where go >nul 2>&1 || (echo [ERROR] Go not found & pause & exit /b 1)
where node >nul 2>&1 || (echo [ERROR] Node not found & pause & exit /b 1)

if not exist "backend\config.yaml" (
  copy /Y "backend\config.example.yaml" "backend\config.yaml" >nul
  echo [WARN] Edit backend\config.yaml then rerun
  notepad "backend\config.yaml"
  pause
  exit /b 1
)

REM quick start: start.bat quick  (skip rebuild if binary+dist exist)
if /I "%~1"=="quick" goto start_only

echo [1/4] build frontend...
pushd frontend
call npm install --silent
call npm run build
if errorlevel 1 (echo [ERROR] frontend build failed & popd & pause & exit /b 1)
popd

echo [2/4] build backend...
pushd backend
go build -o tdy-server.exe .
if errorlevel 1 (echo [ERROR] backend build failed & popd & pause & exit /b 1)
popd
goto start_server

:start_only
if not exist "backend\tdy-server.exe" (
  echo [ERROR] backend\tdy-server.exe missing, run without quick
  pause
  exit /b 1
)
if not exist "frontend\dist\index.html" (
  echo [ERROR] frontend\dist missing, run without quick
  pause
  exit /b 1
)
echo [quick] skip rebuild

:start_server
echo [3/4] start server...
taskkill /FI "WINDOWTITLE eq TDY-Server*" /T /F >nul 2>&1
start "TDY-Server" /D "%~dp0backend" cmd /k tdy-server.exe

echo [4/4] wait health (max 90s)...
set /a n=0
:waitloop
set /a n+=1
if %n% GTR 45 (
  echo [WARN] health timeout, open browser anyway
  goto open
)
timeout /t 2 /nobreak >nul
powershell -NoProfile -Command "try { $r=Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8080/api/health -TimeoutSec 2; if($r.StatusCode -eq 200){exit 0}else{exit 1} } catch { exit 1 }" >nul 2>&1
if errorlevel 1 goto waitloop

:open
start "" "http://127.0.0.1:8080/"
echo.
echo Ready: http://127.0.0.1:8080/
echo Login: admin / admin123
echo Tip: next time use  start.bat quick  for faster restart
echo Keep window TDY-Server open.
echo.
pause
