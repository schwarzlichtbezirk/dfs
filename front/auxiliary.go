package main

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"google.golang.org/grpc/grpclog"
)

var (
	evlre = regexp.MustCompile(`\$\w+`)     // env var with linux-like syntax
	evure = regexp.MustCompile(`\$\{\w+\}`) // env var with unix-like syntax
	evwre = regexp.MustCompile(`\%\w+\%`)   // env var with windows-like syntax
)

// EnvFmt helps to format path patterns, it expands contained environment variables to there values.
func EnvFmt(p string) string {
	return evwre.ReplaceAllStringFunc(evure.ReplaceAllStringFunc(evlre.ReplaceAllStringFunc(p, func(name string) string {
		// strip $VAR and replace by environment value
		if val, ok := os.LookupEnv(name[1:]); ok {
			return val
		} else {
			return name
		}
	}), func(name string) string {
		// strip ${VAR} and replace by environment value
		if val, ok := os.LookupEnv(name[2 : len(name)-1]); ok {
			return val
		} else {
			return name
		}
	}), func(name string) string {
		// strip %VAR% and replace by environment value
		if val, ok := os.LookupEnv(name[1 : len(name)-1]); ok {
			return val
		} else {
			return name
		}
	})
}

// PathExists makes check up on path existance.
func PathExists(path string) (bool, error) {
	var err error
	if _, err = os.Stat(path); err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return true, err
}

// CheckPath is short variant of path existance check.
func CheckPath(fpath string, fname string) (string, bool) {
	if ok, _ := PathExists(path.Join(fpath, fname)); !ok {
		return "", false
	}
	return fpath, true
}

const cfgbase = "config"

// ConfigPath determines configuration path, depended on what directory is exist.
var ConfigPath string

// ErrNoCongig is "no configuration path was found" error message.
var ErrNoCongig = errors.New("no configuration path was found")

// DetectConfigPath finds configuration path with existing configuration file at least.
func DetectConfigPath() (retpath string, err error) {
	var ok bool
	var fpath string
	var exepath = path.Dir(filepath.ToSlash(os.Args[0]))

	// try to get from environment setting
	if fpath, ok = os.LookupEnv("CONFIGPATH"); ok {
		fpath = filepath.ToSlash(fpath)
		// try to get access to full path
		if retpath, ok = CheckPath(fpath, cfgfile); ok {
			return
		}
		// try to find relative from executable path
		if retpath, ok = CheckPath(path.Join(exepath, fpath), cfgfile); ok {
			return
		}
		grpclog.Warningf("no access to pointed configuration path '%s'\n", fpath)
	}

	// try to get from config subdirectory on executable path
	if retpath, ok = CheckPath(path.Join(exepath, cfgbase), cfgfile); ok {
		return
	}
	// try to find in executable path
	if retpath, ok = CheckPath(exepath, cfgfile); ok {
		return
	}
	// try to find in config subdirectory of current path
	if retpath, ok = CheckPath(cfgbase, cfgfile); ok {
		return
	}
	// try to find in current path
	if retpath, ok = CheckPath(".", cfgfile); ok {
		return
	}
	// check up current path is the git root path
	if retpath, ok = CheckPath(cfgbase, cfgfile); ok {
		return
	}

	// check up running in devcontainer workspace
	if retpath, ok = CheckPath(path.Join("/workspaces", gitname, cfgbase), cfgfile); ok {
		return
	}

	// check up git source path
	if fpath, ok = os.LookupEnv("GOPATH"); ok {
		if retpath, ok = CheckPath(path.Join(filepath.ToSlash(fpath), "src", gitpath, cfgbase), cfgfile); ok {
			return
		}
	}

	// if GOBIN or GOPATH is present
	if fpath, ok = os.LookupEnv("GOBIN"); !ok {
		if fpath, ok = os.LookupEnv("GOPATH"); ok {
			fpath = path.Join(fpath, "bin")
		}
	}
	if ok {
		fpath = filepath.ToSlash(fpath)
		// try to get from go bin config
		if retpath, ok = CheckPath(path.Join(fpath, cfgbase), cfgfile); ok {
			return
		}
		// try to get from go bin root
		if retpath, ok = CheckPath(fpath, cfgfile); ok {
			return
		}
	}

	// no config was found
	err = ErrNoCongig
	return
}
