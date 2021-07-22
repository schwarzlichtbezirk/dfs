package main

import (
	"sync"

	"github.com/schwarzlichtbezirk/dfs/pb"
)

type NodeInfo struct {
	// Client is gRPC client.
	Client pb.DataGuideClient
	// Addr is client address:port, used for read-only after initialization.
	Addr string
	// SumSize is total size of all chunks saved on node, atomic increments.
	SumSize int64
	// NumChunks is number of chunks saved on node.
	NumChunks int
}

// Nodes is list of available nodes with information about them.
var Nodes []NodeInfo

// mutex for Nodes array access.
var nodmux sync.RWMutex

// Storage is singleton, files database with fileID/FileInfo keys/values.
var storage sync.Map

// idconter is file ID counter.
var idconter int64

// FileInfo is file information about chunks placed at nodes.
type FileInfo struct {
	FileID int64       `json:"file_id"`
	Name   string      `json:"name"`
	Size   int64       `json:"size"`
	MIME   string      `json:"mime"`
	Chunks []*pb.Range `json:"chunks"`
}

// NodesAdd adds file information to nodes storage.
func (fi *FileInfo) NodesAdd() {
	// update statistics
	nodmux.Lock()
	for _, rng := range fi.Chunks {
		Nodes[rng.NodeId].NumChunks++
		Nodes[rng.NodeId].SumSize += rng.To - rng.From
	}
	nodmux.Unlock()

	// add itself
	storage.Store(fi.FileID, fi)
}

// NodesDel deletes file information from nodes storage.
func (fi *FileInfo) NodesDel() {
	// delete itself
	storage.Delete(fi.FileID)

	// update statistics
	nodmux.Lock()
	for _, rng := range fi.Chunks {
		Nodes[rng.NodeId].NumChunks--
		Nodes[rng.NodeId].SumSize -= rng.To - rng.From
	}
	nodmux.Unlock()
}
