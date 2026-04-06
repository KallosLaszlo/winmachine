@echo off
setlocal enabledelayedexpansion
title WinMachine Build
echo ============================================
echo   WinMachine Build Script
echo ============================================
echo.

:: ---- Check Go ----
where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go is not installed or not in PATH.
    echo         Download from: https://go.dev/dl/
    echo.
    pause
    exit /b 1
)
for /f "tokens=3" %%v in ('go version') do set GO_VER=%%v
echo [OK] Go found: %GO_VER%

:: ---- Check Node.js ----
where node >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Node.js is not installed or not in PATH.
    echo         Download from: https://nodejs.org/
    echo.
    pause
    exit /b 1
)
for /f "tokens=1" %%v in ('node --version') do set NODE_VER=%%v
echo [OK] Node.js found: %NODE_VER%

:: ---- Check npm ----
where npm >nul 2>&1
if %errorlevel% neq 0 (
    echo [WARN] npm not found in PATH, trying common location...
    set "PATH=C:\Program Files\nodejs;%PATH%"
    where npm >nul 2>&1
    if %errorlevel% neq 0 (
        echo [ERROR] npm is not available. Reinstall Node.js.
        pause
        exit /b 1
    )
)
echo [OK] npm found

:: ---- Check / Install Wails CLI ----
where wails >nul 2>&1
if %errorlevel% neq 0 (
    echo [....] Wails CLI not found. Installing...
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    if %errorlevel% neq 0 (
        echo [ERROR] Failed to install Wails CLI.
        pause
        exit /b 1
    )
    echo [OK] Wails CLI installed
) else (
    echo [OK] Wails CLI found
)

:: ---- Ensure npm PATH is available for wails build ----
set "PATH=C:\Program Files\nodejs;%PATH%"

:: ---- Install frontend dependencies if needed ----
if not exist "frontend\node_modules\" (
    echo [....] Installing frontend dependencies...
    pushd frontend
    call npm install
    if %errorlevel% neq 0 (
        echo [ERROR] npm install failed.
        popd
        pause
        exit /b 1
    )
    popd
    echo [OK] Frontend dependencies installed
) else (
    echo [OK] Frontend dependencies present
)

:: ---- Install Go dependencies ----
echo [....] Syncing Go modules...
go mod tidy
if %errorlevel% neq 0 (
    echo [ERROR] go mod tidy failed.
    pause
    exit /b 1
)
echo [OK] Go modules ready

:: ---- Build ----
echo.
echo [....] Building WinMachine...
echo.
wails build
if %errorlevel% neq 0 (
    echo.
    echo [ERROR] Build failed!
    pause
    exit /b 1
)

echo.
echo ============================================
echo   Build successful!
echo   Output: build\bin\WinMachine.exe
echo ============================================
echo.
pause
