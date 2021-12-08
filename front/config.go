package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
)

// CfgCmdLine is command line arguments representation for YAML settings.
type CfgCmdLine struct {
	ConfigPath string `json:"config-path" yaml:"config-path" env:"CONFIGPATH" short:"c" long:"cfgpath" description:"configuration path"`
}

// CfgWebServ is web server settings.
type CfgWebServ struct {
	PortHTTP          []string      `json:"port-http" yaml:"port-http"`
	ReadTimeout       time.Duration `json:"read-timeout" yaml:"read-timeout"`
	ReadHeaderTimeout time.Duration `json:"read-header-timeout" yaml:"read-header-timeout"`
	WriteTimeout      time.Duration `json:"write-timeout" yaml:"write-timeout"`
	IdleTimeout       time.Duration `json:"idle-timeout" yaml:"idle-timeout"`
	MaxHeaderBytes    int           `json:"max-header-bytes" yaml:"max-header-bytes"`
	// Maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" yaml:"shutdown-timeout"`
}

type CfgStorage struct {
	NodeFluidFill    bool  `json:"node-fluid-fill" yaml:"node-fluid-fill"`
	MinNodeChunkSize int64 `json:"min-node-chunk-size" yaml:"min-node-chunk-size"`
	StreamChunkSize  int64 `json:"stream-chunk-size" yaml:"stream-chunk-size"`
}

// Config is common service settings.
type Config struct {
	CfgCmdLine `json:"command-line" yaml:"command-line"`
	CfgWebServ `json:"webserver" yaml:"webserver"`
	CfgStorage `json:"storage" yaml:"storage"`
}

// cfg is instance of common service settings.
var cfg = Config{ // inits default values:
	CfgWebServ: CfgWebServ{
		PortHTTP:          []string{":8010"},
		ReadTimeout:       time.Duration(15) * time.Second,
		ReadHeaderTimeout: time.Duration(15) * time.Second,
		WriteTimeout:      time.Duration(15) * time.Second,
		IdleTimeout:       time.Duration(60) * time.Second,
		MaxHeaderBytes:    1 << 20,
		ShutdownTimeout:   time.Duration(15) * time.Second,
	},
	CfgStorage: CfgStorage{
		NodeFluidFill:    true,
		MinNodeChunkSize: 4 * 1024,
		StreamChunkSize:  1024,
	},
}

func init() {
	if _, err := flags.Parse(&cfg); err != nil {
		os.Exit(1)
	}
}

const (
	gitname = "dfs"
	cfgfile = gitname + "-front.yaml"
	cfgbase = gitname + "-config"

	nodesfile = gitname + "-nodes.yaml"
)

// ConfigPath determines configuration path, depended on what directory is exist.
var ConfigPath string

// ErrNoCongig is "no configuration path was found" error message.
var ErrNoCongig = errors.New("no configuration path was found")

// DetectConfigPath finds configuration path with existing configuration file at least.
func DetectConfigPath() (retpath string, err error) {
	var ok bool
	var path string
	var exepath = filepath.Dir(os.Args[0])

	// try to get from environment setting
	if cfg.ConfigPath != "" {
		path = envfmt(cfg.ConfigPath)
		// try to get access to full path
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
		// try to find relative from executable path
		path = filepath.Join(exepath, path)
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
		log.Printf("no access to pointed configuration path '%s'\n", cfg.ConfigPath)
	}

	// try to get from config subdirectory on executable path
	path = filepath.Join(exepath, cfgbase)
	if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
		retpath = path
		return
	}
	// try to find in executable path
	if ok, _ = pathexists(filepath.Join(exepath, cfgfile)); ok {
		retpath = exepath
		return
	}
	// try to find in current path
	if ok, _ = pathexists(cfgfile); ok {
		retpath = "."
		return
	}
	// try to find config in current path
	if ok, _ = pathexists(filepath.Join(cfgbase, cfgfile)); ok {
		retpath = cfgbase
		return
	}
	// check up current path is the git root path
	if ok, _ = pathexists(filepath.Join("config", cfgfile)); ok {
		retpath = "config"
		return
	}
	// check up running in devcontainer workspace
	path = filepath.Join("/workspaces", gitname, "config")
	if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
		retpath = path
		return
	}

	// if GOBIN or GOPATH is present
	var gobin string
	if gobin, ok = os.LookupEnv("GOBIN"); !ok {
		if gobin, ok = os.LookupEnv("GOPATH"); ok {
			gobin = filepath.Join(gobin, "bin")
		}
	}
	if ok {
		// try to get from go bin config
		path = filepath.Join(gobin, cfgbase)
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			return
		}
		// try to get from go bin root
		if ok, _ = pathexists(filepath.Join(gobin, cfgfile)); ok {
			retpath = gobin
			return
		}
	}

	// no config was found
	err = ErrNoCongig
	return
}

// ReadYaml reads "data" object from YAML-file with given file path.
func ReadYaml(fname string, data interface{}) (err error) {
	var body []byte
	if body, err = os.ReadFile(filepath.Join(ConfigPath, fname)); err != nil {
		return
	}
	if err = yaml.Unmarshal(body, data); err != nil {
		return
	}
	return
}
