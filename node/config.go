package main

import (
	"errors"
	"flag"
	"os"
)

// Config is common service settings.
type Config struct {
	PortGRPC string `json:"port-grpc" yaml:"port-grpc"`
}

// Command line parameters
var (
	portgrpc = flag.String("p", "", "port used by this node for gRPC exchange")
)

// Instance of common service settings.
var cfg = Config{ // inits default values:
	PortGRPC: ":50051",
}

// ErrNoPorts is "no ports were given to node" error message.
var ErrNoPorts = errors.New("no ports were given to node")

// DetectPort explores incoming data sources for port that will be used for gRPC.
func DetectPort() (err error) {
	if envport := os.Getenv("DFSNODEPORT"); len(envport) > 0 {
		cfg.PortGRPC = ":" + envport
		return
	}

	if len(*portgrpc) > 0 {
		cfg.PortGRPC = *portgrpc
		return
	}

	return ErrNoPorts
}
