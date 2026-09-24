@echo off
setlocal EnableExtensions
cd /d "%~dp0.."

set OUT=dist-release
set APP=%OUT%\tdy-manager

echo ========================================
echo   Pack Linux release (amd64)
echo ========================================

if exist "%OUT%" rmdir /s /q "%OUT%"
mkdir "%APP%\static" 2>nul
mkdir "%APP%\uploads" 2>nul

echo [1/3] build frontend...
pushd frontend
call npm install --silent
call npm run build
if errorlevel 1 (echo [ERROR] frontend build failed & popd & exit /b 1)
popd
xcopy /E /I /Y frontend\dist\* "%APP%\static\" >nul

echo [2/3] build linux backend...
pushd backend
set CGO_ENABLED=0
set GOOS=linux
set GOARCH=amd64
go build -ldflags="-s -w" -o "..\%APP%\tdy-server" .
if errorlevel 1 (echo [ERROR] go build failed & popd & exit /b 1)
popd

echo [3/3] copy deploy files...
copy /Y deploy\config.production.yaml.example "%APP%\config.yaml.example" >nul
copy /Y deploy\tdy-manager.service "%APP%\" >nul
copy /Y deploy\nginx.conf.example "%APP%\" >nul
copy /Y deploy\install.sh "%APP%\" >nul

echo.
echo OK: %CD%\%APP%
echo.
echo Next (PowerShell):
echo   scp -r dist-release\tdy-manager user@SERVER:/tmp/
echo   ssh user@SERVER "sudo bash /tmp/tdy-manager/install.sh"
echo.
echo Or zip first:
echo   tar -czf dist-release\tdy-manager-linux-amd64.tar.gz -C dist-release tdy-manager
pause
