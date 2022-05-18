@echo off

set wsdir=%~dp0..
set pbdir=%wsdir%/pb
set expdir=%wsdir%/api/export
set impdir=%wsdir%/api/import

protoc -I=%expdir% -I=%impdir%^
 --go_out %pbdir%^
 --go_opt paths=source_relative^
 --go-grpc_out %pbdir%^
 --go-grpc_opt paths=source_relative^
 dfs.proto

protoc -I=%expdir% -I=%impdir% -I=%pbdir%^
 --gotag_out outdir=./pb:.^
 --gotag_opt auto="yaml-as-snake+xml-as-snake"^
 --gotag_opt paths=source_relative^
 dfs.proto
