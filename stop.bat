@echo off
echo Stopping TDY-Server...
taskkill /FI "WINDOWTITLE eq TDY-Server*" /T /F >nul 2>&1
taskkill /FI "WINDOWTITLE eq TDY-Backend*" /T /F >nul 2>&1
taskkill /FI "WINDOWTITLE eq TDY-Frontend*" /T /F >nul 2>&1
echo Done.
pause
