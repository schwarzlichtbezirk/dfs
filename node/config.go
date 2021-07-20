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

// Instance of common service settings.
var cfg = Config{ // inits default values:
	PortGRPC: ":50050",
}

var ErrNoPorts = errors.New("no ports were given to node")

func DetectPort() (err error) {
	if cfg.PortGRPC = os.Getenv("DFSNODEPORT"); len(cfg.PortGRPC) > 0 {
		return
	}

	flag.Parse()
	flag.StringVar(&cfg.PortGRPC, "p", "", "port used by this node for gRPC exchange")
	if len(cfg.PortGRPC) > 0 {
		return
	}

	return ErrNoPorts
}
