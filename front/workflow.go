package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	// channel to indicate about server shutdown
	exitchan = make(chan struct{})
	// wait group for all server goroutines
	exitwg sync.WaitGroup
	// wait group for grpc goroutines
	grpcwg sync.WaitGroup
)

// Run launches server listeners.
func Run(gmux *Router) {
	var err error

	// get confiruration path
	if err = DetectConfigPath(); err != nil {
		log.Fatal(err)
	}
	log.Printf("config path: %s\n", ConfigPath)

	// load content of Config structure from YAML-file.
	if err = ReadYaml(cfgfile, &cfg); err != nil {
		log.Fatalf("can not read '%s' file: %v\n", cfgfile, err)
	}
	log.Printf("loaded '%s'\n", cfgfile)

	var nodes []string
	if err = ReadYaml(nodesfile, &nodes); err != nil {
		log.Fatal(err)
	}
	log.Printf("total %d nodes\n", len(nodes))
	storage.Nodes = make([]*NodeInfo, len(nodes))

	// starts gRPC clients
	for i, addr := range nodes {
		var node = &NodeInfo{
			Addr:    addr,
			SumSize: 0,
		}
		storage.Nodes[i] = node
		node.RunGRPC()
	}
	// wait until all grpc starts
	grpcwg.Wait()

	// check on exit during grpc connecting
	select {
	case <-exitchan:
		return
	default:
	}

	// starts HTTP servers
	for _, addr := range cfg.PortHTTP {
		var addr = envfmt(addr) // localize
		exitwg.Add(1)
		go func() {
			defer exitwg.Done()

			var server = &http.Server{
				Addr:              addr,
				Handler:           gmux,
				ReadTimeout:       cfg.ReadTimeout,
				ReadHeaderTimeout: cfg.ReadHeaderTimeout,
				WriteTimeout:      cfg.WriteTimeout,
				IdleTimeout:       cfg.IdleTimeout,
				MaxHeaderBytes:    cfg.MaxHeaderBytes,
			}
			go func() {
				log.Printf("start http on %s\n", addr)
				if err := server.ListenAndServe(); err != http.ErrServerClosed {
					log.Fatalf("failed to serve: %v", err)
				}
			}()

			// wait for exit signal
			<-exitchan

			// create a deadline to wait for.
			var ctx, cancel = context.WithTimeout(
				context.Background(),
				cfg.ShutdownTimeout)
			defer cancel()

			server.SetKeepAlivesEnabled(false)
			if err := server.Shutdown(ctx); err != nil {
				log.Printf("shutdown http on %s: %v\n", addr, err)
			} else {
				log.Printf("stop http on %s\n", addr)
			}
		}()
	}

	log.Printf("ready")
}

// WaitBreak blocks goroutine until Ctrl+C will be pressed.
func WaitBreak() {
	var sigint = make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM (Ctrl+/)
	// SIGKILL, SIGQUIT will not be caught.
	signal.Notify(sigint, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Block until we receive our signal.
	<-sigint
	// Make exit signal.
	close(exitchan)
}

// WaitExit performs graceful network shutdown,
// waits until all server threads will be stopped.
func WaitExit() {
	exitwg.Wait()
}
