package main

import (
	"errors"
	"flag"
	"os"
	"strings"
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

// ErrNoPorts is "no port was given to node" error message.
var ErrNoPort = errors.New("no port was given to node")

// DetectPort explores incoming data sources for port that will be used for gRPC.
func DetectPort() (err error) {
	defer func() {
		if err == nil {
			if !strings.HasPrefix(cfg.PortGRPC, ":") {
				cfg.PortGRPC = ":" + cfg.PortGRPC
			}
		}
	}()

	if envport := os.Getenv("NODEPORT"); len(envport) > 0 {
		cfg.PortGRPC = envport
		return
	}
	if len(*portgrpc) > 0 {
		cfg.PortGRPC = *portgrpc
		return
	}

	return ErrNoPort
}
