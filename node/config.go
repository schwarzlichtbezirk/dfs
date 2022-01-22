package main

import (
	"os"
	"strings"

	"github.com/jessevdk/go-flags"
)

// Instance of common service settings.
var cfg struct {
	PortGRPC string `json:"port-grpc" yaml:"port-grpc" env:"NODEPORT" short:"p" long:"portgrpc" default:":50051" description:"Port used by this node for gRPC exchange."`
}

// compiled binary version, sets by compiler with command
//    go build -ldflags="-X 'main.buildvers=%buildvers%'"
var buildvers string

// compiled binary build date, sets by compiler with command
//    go build -ldflags="-X 'main.builddate=%date%'"
var builddate string

func init() {
	if _, err := flags.Parse(&cfg); err != nil {
		os.Exit(1)
	}
	// correct config
	if !strings.HasPrefix(cfg.PortGRPC, ":") {
		cfg.PortGRPC = ":" + cfg.PortGRPC
	}
}
