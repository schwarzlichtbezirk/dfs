package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/jessevdk/go-flags"
	"google.golang.org/grpc/grpclog"
)

var (
	// context to indicate about service shutdown
	exitctx context.Context
	exitfn  context.CancelFunc
	// wait group for all server goroutines
	exitwg sync.WaitGroup
	// wait group for grpc goroutines
	grpcwg sync.WaitGroup
)

func init() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stderr, os.Stderr))
}

// Init performs global data initialization.
func Init() {
	grpclog.Infof("version: %s, builton: %s\n", buildvers, builddate)
	grpclog.Infoln("starts")

	// create context and wait the break
	exitctx, exitfn = context.WithCancel(context.Background())
	go func() {
		// Make exit signal on function exit.
		defer exitfn()

		var sigint = make(chan os.Signal, 1)
		var sigterm = make(chan os.Signal, 1)
		// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C) or SIGTERM (Ctrl+/)
		// SIGKILL, SIGQUIT will not be caught.
		signal.Notify(sigint, syscall.SIGINT)
		signal.Notify(sigterm, syscall.SIGTERM)
		// Block until we receive our signal.
		select {
		case <-exitctx.Done():
			if errors.Is(exitctx.Err(), context.DeadlineExceeded) {
				grpclog.Infoln("shutting down by timeout")
			} else if errors.Is(exitctx.Err(), context.Canceled) {
				grpclog.Infoln("shutting down by cancel")
			} else {
				grpclog.Infoln("shutting down by %s", exitctx.Err().Error())
			}
		case <-sigint:
			grpclog.Infoln("shutting down by break")
		case <-sigterm:
			grpclog.Infoln("shutting down by process termination")
		}
		signal.Stop(sigint)
		signal.Stop(sigterm)
	}()

	// load content of Config structure from YAML-file.
	var err error

	// get confiruration path
	if ConfigPath, err = DetectConfigPath(); err != nil {
		grpclog.Fatal(err)
	}
	grpclog.Infof("config path: %s\n", ConfigPath)

	log.Println(cfg.PortHTTP)
	if err = ReadYaml(cfgfile, &cfg); err != nil {
		grpclog.Fatalf("can not read '%s' file: %v\n", cfgfile, err)
	}
	grpclog.Infof("loaded '%s'\n", cfgfile)
	log.Println(cfg.PortHTTP)
	// second iteration, rewrite settings from config file
	if _, err = flags.NewParser(&cfg, flags.PassDoubleDash).Parse(); err != nil {
		panic("no way to here")
	}

	// correct config
	if cfg.MinNodeChunkSize <= 0 {
		cfg.MinNodeChunkSize = 4 * 1024
		grpclog.Warningf("'min-node-chunk-size' is adjusted to %d\n", cfg.MinNodeChunkSize)
	}
	if cfg.StreamChunkSize <= 0 {
		cfg.StreamChunkSize = 512
		grpclog.Warningf("'stream-chunk-size' is adjusted to %d\n", cfg.StreamChunkSize)
	}
	grpclog.Infof("expects %d nodes\n", len(cfg.NodeList))
	storage.Nodes = make([]*NodeInfo, len(cfg.NodeList))
}

// Run launches server listeners.
func Run(gmux *Router) {
	// starts gRPC clients
	for i, addr := range cfg.NodeList {
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
	case <-exitctx.Done():
		return
	default:
	}

	// starts HTTP listeners
	var httpwg sync.WaitGroup
	for _, addr := range cfg.PortHTTP {
		var addr = EnvFmt(addr) // localize
		httpwg.Add(1)
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

			grpclog.Infof("start http on %s\n", addr)
			go func() {
				httpwg.Done()
				if err := server.ListenAndServe(); err != http.ErrServerClosed {
					grpclog.Fatalf("failed to serve at all: %v", err)
				}
			}()

			// wait for exit signal
			<-exitctx.Done()

			// create a deadline to wait for.
			var ctx, cancel = context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
			defer cancel()

			server.SetKeepAlivesEnabled(false)
			if err := server.Shutdown(ctx); err != nil {
				grpclog.Errorf("shutdown http on %s: %v\n", addr, err)
			} else {
				grpclog.Infof("stop http on %s\n", addr)
			}
		}()
	}
	httpwg.Wait()
	grpclog.Infoln("service ready")
}

// Done performs graceful network shutdown,
// waits until all server threads will be stopped.
func Done() {
	// wait for exit signal
	<-exitctx.Done()
	// wait until all server threads will be stopped.
	exitwg.Wait()
	grpclog.Infoln("shutting down complete.")
}
