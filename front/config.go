package main

import (
	"os"
	"path"
	"time"

	"github.com/jessevdk/go-flags"
	"gopkg.in/yaml.v3"
)

const (
	gitname = "dfs"
	gitpath = "github.com/schwarzlichtbezirk/" + gitname
	cfgfile = "dfs-front.yaml"
)

// CfgWebServ is web server settings.
type CfgWebServ struct {
	PortHTTP          []string      `json:"port-http" yaml:"port-http" env:"PORTHTTP" env-delim:";" short:"w" long:"http" description:"List of address:port values for non-encrypted connections. Address is skipped in most common cases, port only remains."`
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
	ApiTimeout       time.Duration `json:"api-timeout" yaml:"api-timeout" long:"at" description:"gRPC API call timeout."`
}

// Config is common service settings.
type Config struct {
	CfgWebServ `json:"web-server" yaml:"web-server" group:"Web Server"`
	CfgStorage `json:"storage" yaml:"storage" group:"Storage"`
	// list of nodes
	NodeList []string `json:"node-list" yaml:"node-list" env:"NODELIST" env-delim:";" short:"n" long:"node" description:"Distributed file server list of nodes."`
}

// cfg is instance of common service settings.
var cfg = Config{ // inits default values:
	CfgWebServ: CfgWebServ{
		PortHTTP:          []string{":8008", ":8010"},
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
	NodeList: []string{"localhost:50051", "localhost:50052"},
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
}

// ReadYaml reads "data" object from YAML-file with given file name.
func ReadYaml(fname string, data interface{}) (err error) {
	var body []byte
	if body, err = os.ReadFile(path.Join(ConfigPath, fname)); err != nil {
		return
	}
	if err = yaml.Unmarshal(body, data); err != nil {
		return
	}
	return
}
