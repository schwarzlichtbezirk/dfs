#!/bin/bash
cd $GOPATH/src/github.com/schwarzlichtbezirk/dfs
mkdir -pv $GOPATH/bin/dfs-config
cp -ruv ./config/* $GOPATH/bin/dfs-config
go env -w GOOS=linux GOARCH=amd64
go build -o $GOPATH/bin/dfs.front.x64 -v ./front
go build -o $GOPATH/bin/dfs.node.x64 -v ./node
