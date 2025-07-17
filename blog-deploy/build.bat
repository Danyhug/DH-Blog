@echo off
setlocal enabledelayedexpansion

REM 变量
set BINARY_NAME=dhblog
set BACKEND_DIR=..\blog-backend
set FRONTEND_DIR=..\blog-front
set DEPLOY_DIR=%CD%\build
set FRONT_TEMP_DIR=%CD%\front
set EMBED_DIR=%BACKEND_DIR%\internal\frontend\dist

REM 1. 准备目录
if exist "%DEPLOY_DIR%" rmdir /s /q "%DEPLOY_DIR%"
if exist "%FRONT_TEMP_DIR%" rmdir /s /q "%FRONT_TEMP_DIR%"
if exist "%EMBED_DIR%" rmdir /s /q "%EMBED_DIR%"
mkdir "%DEPLOY_DIR%"

REM 2. 构建前端
echo 构建前端...
cd /d "%FRONTEND_DIR%"
call pnpm build || goto :fail
move /Y dist "%FRONT_TEMP_DIR%"
cd /d "%~dp0"

REM 3. 嵌入前端到后端
echo 准备后端嵌入前端文件...
mkdir "%EMBED_DIR%"
xcopy /E /I /Y "%FRONT_TEMP_DIR%\*" "%EMBED_DIR%\"

REM 4. 多平台构建
call :build_for_platform windows amd64 -windows-amd64.exe
call :build_for_platform linux amd64 -linux-amd64
REM Windows 下无法直接交叉编译 macOS 版本，需在 macOS/Linux 下编译
REM call :build_for_platform darwin arm64 -darwin-arm64

REM 5. 清理中间文件
echo 清理中间文件...
if exist "%FRONT_TEMP_DIR%" rmdir /s /q "%FRONT_TEMP_DIR%"
if exist "%EMBED_DIR%" rmdir /s /q "%EMBED_DIR%"

echo 全部构建完成！产物在 %DEPLOY_DIR%
goto :eof

:build_for_platform
set OS=%1
set ARCH=%2
set EXT=%3
set OUTPUT=%DEPLOY_DIR%\%BINARY_NAME%%EXT%
echo 构建 %OS%/%ARCH% ...
cd /d "%BACKEND_DIR%"
set GOOS=%OS%
set GOARCH=%ARCH%
go build -ldflags="-s -w" -o "%OUTPUT%" ./cmd/blog-backend
if errorlevel 1 (
    echo ❌ %OS%/%ARCH% 构建失败
    exit /b 1
) else (
    echo ✅ %OUTPUT%
)
cd /d "%~dp0"
goto :eof

:fail
echo 构建失败！
exit /b 1 