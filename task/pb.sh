#!/bin/bash

wsdir=$(dirname $0)/..
pbdir=$wsdir/pb
apidir=$wsdir/api/export

protoc -I=$apidir\
 --go_out $pbdir\
 --go_opt paths=source_relative\
 --go-grpc_out $pbdir\
 --go-grpc_opt paths=source_relative\
 dfs.proto
