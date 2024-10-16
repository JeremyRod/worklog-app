@echo off
set VERSION=1.0.3
for /f %%i in ('git rev-parse --short HEAD') do set GIT_COMMIT=%%i

go build -ldflags "-X main.version=%VERSION% -X main.gitCommit=%GIT_COMMIT%" -o worklog.exe