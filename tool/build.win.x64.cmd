@echo off
cd /d %~dp0..
rem puts version to file for docker image building
git describe --tags > buildvers.txt
set /p buildvers=<buildvers.txt
set builddate="%date%"
xcopy .\config %GOPATH%\bin\config /f /d /i /e /k /y
go env -w GOOS=windows GOARCH=amd64
go build -o %GOPATH%/bin/dfs.front.x64.exe -v -ldflags="-X 'main.buildvers=%buildvers%' -X 'main.builddate=%builddate%'" ./front
go build -o %GOPATH%/bin/dfs.node.x64.exe -v -ldflags="-X 'main.buildvers=%buildvers%' -X 'main.builddate=%builddate%'" ./node
