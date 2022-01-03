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
	ConfigPath string `json:"-" yaml:"-" env:"CONFIGPATH" short:"c" long:"cfgpath" description:"Configuration path. Can be full path to config folder, or relative from executable destination."`
	NoConfig   bool   `json:"-" yaml:"-" long:"nocfg" description:"Specifies do not load settings from YAML-settings file, keeps default."`
}

// CfgWebServ is web server settings.
type CfgWebServ struct {
	PortHTTP          []string      `json:"port-http" yaml:"port-http" short:"w" long:"porthttp" description:"List of address:port values for non-encrypted connections. Address is skipped in most common cases, port only remains."`
	ReadTimeout       time.Duration `json:"read-timeout" yaml:"read-timeout" long:"rt" description:"Maximum duration for reading the entire request, including the body."`
	ReadHeaderTimeout time.Duration `json:"read-header-timeout" yaml:"read-header-timeout" long:"rht" description:"Amount of time allowed to read request headers."`
	WriteTimeout      time.Duration `json:"write-timeout" yaml:"write-timeout" long:"wt" description:"Maximum duration before timing out writes of the response."`
	IdleTimeout       time.Duration `json:"idle-timeout" yaml:"idle-timeout" long:"it" description:"Maximum amount of time to wait for the next request when keep-alives are enabled."`
	MaxHeaderBytes    int           `json:"max-header-bytes" yaml:"max-header-bytes" long:"mhb" description:"Controls the maximum number of bytes the server will read parsing the request header's keys and values, including the request line, in bytes."`
	// Maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" yaml:"shutdown-timeout" long:"st" description:"Maximum duration to wait for graceful shutdown."`
}

type CfgStorage struct {
	NodeFluidFill    bool          `json:"node-fluid-fill" yaml:"node-fluid-fill" long:"nff" description:"Points to fill nodes by fluid algorithm."`
	MinNodeChunkSize int64         `json:"min-node-chunk-size" yaml:"min-node-chunk-size" long:"mncs" description:"Minimum size of chunk to divide the file and put to nodes, except last chunk."`
	StreamChunkSize  int64         `json:"stream-chunk-size" yaml:"stream-chunk-size" long:"scs" description:"Maximum chunk size to send to each node during the streaming."`
	ApiTimeout       time.Duration `json:"api-timeout" yaml:"api-timeout"`
}

// Config is common service settings.
type Config struct {
	CfgCmdLine `json:"command-line" yaml:"command-line" group:"Data Parameters"`
	CfgWebServ `json:"webserver" yaml:"webserver" group:"Web Server"`
	CfgStorage `json:"storage" yaml:"storage" group:"Storage"`
	// list of nodes
	NodeList []string `json:"node-list" yaml:"node-list" env:"NODELIST" env-delim:";" short:"n" long:"node" description:"Distributed file server list of nodes."`
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
		ApiTimeout:       2 * time.Second,
	},
}

func init() {
	if _, err := flags.Parse(&cfg); err != nil {
		os.Exit(1)
	}
}

const (
	gitname = "dfs"
	gitpath = "github.com/schwarzlichtbezirk/" + gitname
	cfgbase = gitname + "-config"
	cfgfile = "front.yaml"
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
	// try to find in config subdirectory of current path
	if ok, _ = pathexists(filepath.Join(cfgbase, cfgfile)); ok {
		retpath = cfgbase
		return
	}
	// try to find in current path
	if ok, _ = pathexists(cfgfile); ok {
		retpath = "."
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

	// check up git source path
	var prefix string
	if prefix, ok = os.LookupEnv("GOPATH"); ok {
		path = filepath.Join(prefix, "src", gitpath, "config")
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
	}

	// if GOBIN or GOPATH is present
	if prefix, ok = os.LookupEnv("GOBIN"); !ok {
		if prefix, ok = os.LookupEnv("GOPATH"); ok {
			prefix = filepath.Join(prefix, "bin")
		}
	}
	if ok {
		// try to get from go bin config
		path = filepath.Join(prefix, cfgbase)
		if ok, _ = pathexists(filepath.Join(path, cfgfile)); ok {
			retpath = path
			return
		}
		// try to get from go bin root
		if ok, _ = pathexists(filepath.Join(prefix, cfgfile)); ok {
			retpath = prefix
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
