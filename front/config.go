package main

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// CfgServ is web server settings.
type CfgServ struct {
	PortHTTP          []string      `json:"port-http" yaml:"port-http"`
	ReadTimeout       time.Duration `json:"read-timeout" yaml:"read-timeout"`
	ReadHeaderTimeout time.Duration `json:"read-header-timeout" yaml:"read-header-timeout"`
	WriteTimeout      time.Duration `json:"write-timeout" yaml:"write-timeout"`
	IdleTimeout       time.Duration `json:"idle-timeout" yaml:"idle-timeout"`
	MaxHeaderBytes    int           `json:"max-header-bytes" yaml:"max-header-bytes"`
	// Maximum duration to wait for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown-timeout" yaml:"shutdown-timeout"`
}

// Config is common service settings.
type Config struct {
	CfgServ `json:"webserver" yaml:"webserver"`
}

// cfg is instance of common service settings.
var cfg = Config{ // inits default values:
	CfgServ: CfgServ{
		PortHTTP:          []string{":8010"},
		ReadTimeout:       time.Duration(15) * time.Second,
		ReadHeaderTimeout: time.Duration(15) * time.Second,
		WriteTimeout:      time.Duration(15) * time.Second,
		IdleTimeout:       time.Duration(60) * time.Second,
		MaxHeaderBytes:    1 << 20,
		ShutdownTimeout:   time.Duration(15) * time.Second,
	},
}

const cfgfile = "dfs-fe.yaml"

// ConfigPath determines configuration path, depended on what directory is exist.
var ConfigPath string

var ErrNoCongig = errors.New("no configuration path was found")

// DetectConfigPath finds configuration path.
func DetectConfigPath() (err error) {
	var path string
	// try to get from environment setting
	if path = os.Getenv("APPCONFIGPATH"); path != "" {
		if ok, _ := pathexists(path); ok {
			ConfigPath = path
			return
		}
		log.Printf("no access to pointed configuration path '%s'\n", path)
	}
	// try to get from config subdirectory on executable path
	var exepath = filepath.Dir(os.Args[0])
	path = filepath.Join(exepath, "dfs-config")
	if ok, _ := pathexists(path); ok {
		ConfigPath = path
		return
	}
	// try to find in executable path
	if ok, _ := pathexists(filepath.Join(exepath, cfgfile)); ok {
		ConfigPath = exepath
		return
	}
	// try to find in current path
	if ok, _ := pathexists(cfgfile); ok {
		ConfigPath = "."
		return
	}

	// if GOPATH is present
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		// try to get from go bin config
		path = filepath.Join(gopath, "bin/dfs-config")
		if ok, _ := pathexists(path); ok {
			ConfigPath = path
			return
		}
		// try to get from go bin root
		path = filepath.Join(gopath, "bin")
		if ok, _ := pathexists(filepath.Join(path, cfgfile)); ok {
			ConfigPath = path
			return
		}
		// try to get from source code
		path = filepath.Join(gopath, "src/github.com/schwarzlichtbezirk/dfs/config")
		if ok, _ := pathexists(path); ok {
			ConfigPath = path
			return
		}
	}

	// no config was found
	return ErrNoCongig
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

var efre = regexp.MustCompile(`\$\{\w+\}`)

func envfmt(p string) string {
	return filepath.ToSlash(efre.ReplaceAllStringFunc(p, func(name string) string {
		return os.Getenv(name[2 : len(name)-1]) // strip ${...} and replace by env value
	}))
}

func pathexists(path string) (bool, error) {
	var err error
	if _, err = os.Stat(path); err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
