
# dfs

distributed file server

## Architecture

There has fronend service (named as `front`) and backend node service (named as `node`). In running composition front starts in one instance, and nodes starts in several instances. Conversations between front and nodes are by gRPC, where front is single client and nodes are group of servers for one client. New nodes can be added to running composition.

Front have REST API, and can be accessed outside of composition by this API.

## How to run on localhost

1. First of all install [Golang](https://go.dev/dl/) of last version. Requires that [GOPATH is set](https://golang.org/doc/code.html#GOPATH).

2. Fetch golang `grpc` library.

```batch
go get -u google.golang.org/grpc
```

Note: if there is no access to `golang.org` host, use VPN (via Netherlands/USA) or git repositories cloning.

3. Fetch this source code and compile application.

```batch
go get github.com/schwarzlichtbezirk/dfs
```

Folder `github.com\schwarzlichtbezirk\dfs\task` contains batch helpers to compile services for Windows for x86 and amd64 platforms.

4. Edit config-file `github.com/schwarzlichtbezirk/dfs/config/dfs-nodes.yaml` with addresses of expected nodes on front startup.

5. Run services.

```batch
start "DFS front" %GOPATH%/bin/dfs.front.x64.exe
start "DFS node#1" %GOPATH%/bin/dfs.node.x64.exe -p :50051
start "DFS node#2" %GOPATH%/bin/dfs.node.x64.exe -p :50052
rem and start other nodes instances
```

or run `github.com\schwarzlichtbezirk\dfs\task\start.x64.cmd` batch-file to start composition for default nodes list.

## How to run in docker

1. Change current directory to project root.

```batch
cd /d %GOPATH%/src/github.com/schwarzlichtbezirk/dfs
```

2. Build docker images for `front` and for `node` services.

```batch
docker build --pull --rm -f "node.dockerfile" -t dfs-node:latest "."
docker build --pull --rm -f "front.dockerfile" -t dfs-front:latest "."
```

3. Then run docker compose file.

```batch
docker-compose -f "docker-compose.yaml" up -d --build
```

## What its need else to modify code

If you want to modify `.go`-code and `.proto` file, you should [download](https://github.com/protocolbuffers/protobuf/blob/master/README.md#protocol-compiler-installation) and install protocol buffer compiler. Then install protocol buffer compiler plugins:

```batch
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

To generate protocol buffer code, run `task/pb.cmd` batch file.

## REST API

Arguments of all API calls placed as JSON-objects at request body. Replies comes also only as JSON-objects in all cases.

Errors comes on replies with status >= 300 as objects like `{"what":"some error message","when":1613251727492,"code":3}` where `when` is Unix time in milliseconds of error occurrence, `code` is unique error source point code.

REST API can be tested by `curl` tool. There is no need to write some module that performs curl's job. In followed samples URI address must be replaced to IP or host where service is running.

### Detect used volume on each node

```batch
curl -X GET localhost:8008/api/v0/nodesize
```

Returns integers array with node total data size in each value. Index of each value in array fits to node index.

### Upload file

```batch
curl -i -X POST -H "Content-Type: multipart/form-data" -F "datafile=@H:\src\IMG_20200519_145112.jpg" localhost:8008/api/v0/upload
```

`datafile` here can be some other valid destination path to file.
Application architecture allows uploading multiple files with the same name. It can be same file, or some files with different content and same file name. Each uploaded file gets unique file ID. Returns array of chunks properties.

### Download file

To view previous uploaded image in browser, follow those URL:
<http://localhost:8010/api/v0/download?id=1>
or
<http://localhost:8010/api/v0/download?name=IMG_20200519_145112.jpg>

### Get information about file chunks

```batch
curl -X GET localhost:8008/api/v0/fileinfo -d "{\"id\":1}"
```

or

```batch
curl -X GET localhost:8008/api/v0/fileinfo -d "{\"name\":\"IMG_20200519_145112.jpg\"}"
```

Returns array of chunks properties for file with given `id` or given `name`. Since there can be multiple files uploaded with the same name, and if `name` is pointed, it returns properties of first founded file with given name. Returns `null` if file was not found.

### Remove file from storage

```batch
curl -X POST localhost:8008/api/v0/remove -d "{\"id\":1}"
```

or

```batch
curl -X POST localhost:8008/api/v0/remove -d "{\"name\":\"IMG_20200519_145112.jpg\"}"
```

Deletes all chunks on nodes and information about file with given `id` or given `name`. Returns array of chunks properties of deleted file. Returns `null` if file was not found.

### Add new node at runtime

```batch
curl -X GET localhost:8008/api/v0/addnode -d "{\"addr\":\":50053\"}"
```

Adds new node during service is running. Transaction waits util gRPC connection will be established, and then returns index of added node.

## Simple sample to test the service

1. Upload some first images:

```batch
curl -i -X POST -H "Content-Type: multipart/form-data" -F "datafile=@H:\src\IMG_20200519_145112.jpg" localhost:8010/api/v0/upload
```

2. Check up data volumes used by nodes:

```batch
curl -X GET localhost:8010/api/v0/nodesize
```

3. Start new node instance and add it to composition in runtime:

```batch
start "node#new" dfs.node.x64.exe -p :50080
curl -X POST localhost:8008/api/v0/addnode -d "{\"addr\":\":50080\"}"
```

4. Upload some second image. Percent for new node will be larger:

```batch
curl -i -X POST -H "Content-Type: multipart/form-data" -F "datafile=@H:\src\IMG_20200519_145207.jpg" localhost:8010/api/v0/upload
```

5. Check up data volumes used by nodes again:

```batch
curl -X GET localhost:8010/api/v0/nodesize
```

6. View those images in browser by followed links:

[IMG_20200519_145112.jpg](http://localhost:8010/api/v0/download?id=1) and
[IMG_20200519_145207.jpg](http://localhost:8010/api/v0/download?id=2)

7. Remove 1st image from storage:

```batch
curl -X POST localhost:8010/api/v0/remove -d "{\"id\":1}"
```

8. Check up data volumes again after it:

```batch
curl -X GET localhost:8010/api/v0/nodesize
```

9. Try to get file info for removed file:

```batch
curl -X GET localhost:8010/api/v0/fileinfo -d "{\"id\":1}"
```

10. View file info for second image remaining in storage:

```batch
curl -X GET localhost:8010/api/v0/fileinfo -d "{\"id\":2}"
```

11. Clear data storage:

```batch
curl -X PUT localhost:8010/api/v0/clear
```

12. Check up data volumes is zero:

```batch
curl -X GET localhost:8010/api/v0/nodesize
```

---
(c) schwarzlichtbezirk, 2021.
