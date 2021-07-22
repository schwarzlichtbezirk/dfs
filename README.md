
# dfs

distributed file server


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
curl -i -X POST -H "Content-Type: multipart/form-data" -F "datafile=@D:\imggps\IMG_20181009_141028.jpg" localhost:8008/api/v0/upload
```
`datafile` here can be some other valid destination path to file.
Application architecture allows uploading multiple files with the same name. It can be same file, or some files with different content and same file name. Each uploaded file gets unique file ID. Returns array of chunks properties.


### Get information about file chunks

```batch
curl -X GET localhost:8008/api/v0/fileinfo -d "{\"id\":1}"
```
or
```batch
curl -X GET localhost:8008/api/v0/fileinfo -d "{\"name\":\"IMG_20181009_141028.jpg\"}"
```
Returns array of chunks properties for file with given `id` or given `name`. Since there can be multiple files uploaded with the same name, and if `name` is pointed, it returns properties of first founded file with given name. Returns `null` if file was not found.


### Remove file from storage

```batch
curl -X GET localhost:8008/api/v0/remove -d "{\"id\":1}"
```
or
```batch
curl -X GET localhost:8008/api/v0/remove -d "{\"name\":\"IMG_20181009_141028.jpg\"}"
```
Deletes all chunks on nodes and information about file with given `id` or given `name`. Returns array of chunks properties of deleted file. Returns `null` if file was not found.

---
(c) schwarzlichtbezirk, 2021.
