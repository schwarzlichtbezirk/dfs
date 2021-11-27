@echo off
go env -w GOOS=windows GOARCH=amd64
cd /d  %GOPATH%\src\github.com\schwarzlichtbezirk\dfs
go build -o %GOPATH%/bin/dfs.front.x64.exe -v ./front
go build -o %GOPATH%/bin/dfs.node.x64.exe -v ./node
xcopy .\config %GOPATH%\bin\dfs-config /f /d /i /s /e /k /y
