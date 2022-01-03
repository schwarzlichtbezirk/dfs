@echo off
cd /d %~dp0..
xcopy .\config %GOPATH%\bin\dfs-config /f /d /i /e /k /y
go env -w GOOS=windows GOARCH=amd64
go build -o %GOPATH%/bin/dfs.front.x64.exe -v ./front
go build -o %GOPATH%/bin/dfs.node.x64.exe -v ./node
