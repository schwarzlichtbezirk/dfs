#!/bin/bash
export pbsrc=$GOPATH/src
export pbpkg=github.com/schwarzlichtbezirk/dfs/pb
protoc -I=$pbsrc/$pbpkg --go_out=$pbsrc --go-grpc_out=$pbsrc\
 $pbsrc/$pbpkg/dfs.proto
