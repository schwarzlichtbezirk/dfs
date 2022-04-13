@echo off
cd /d %~dp0..

rem puts version to file for docker image building
git describe --tags > semver
set /p buildvers=<semver
set builddate="%date%"

xcopy .\config %GOPATH%\bin\config /f /d /i /e /k /y
go env -w GOOS=windows GOARCH=386
go build -o %GOPATH%/bin/dfs.front.x86.exe -v -ldflags="-X 'main.buildvers=%buildvers%' -X 'main.builddate=%builddate%'" ./front
go build -o %GOPATH%/bin/dfs.node.x86.exe -v -ldflags="-X 'main.buildvers=%buildvers%' -X 'main.builddate=%builddate%'" ./node
