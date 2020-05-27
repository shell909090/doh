package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"

	logging "github.com/op/go-logging"
	"github.com/shell909090/doh/drivers"
)

const (
	DEFAULT_ALIASES = "doh-aliases.json;~/.doh-aliases.json"
)

var (
	ErrConfigParse        = errors.New("config parse error")
	ErrParameter          = errors.New("parameter error")
	logger                = logging.MustGetLogger("")
	Version        string = "unknown"
)

type Config struct {
	Logfile  string
	Loglevel string
	Service  json.RawMessage
	Client   json.RawMessage
}

func (cfg *Config) CreateClient() (cli drivers.Client) {
	var header drivers.DriverHeader
	err := json.Unmarshal(cfg.Client, &header)
	if err != nil {
		panic(err.Error())
	}
	cli = header.CreateClient(cfg.Client)
	return
}

func (cfg *Config) CreateService(cli drivers.Client) (srv drivers.Server) {
	var header drivers.DriverHeader
	err := json.Unmarshal(cfg.Service, &header)
	if err != nil {
		panic(err.Error())
	}
	srv = header.CreateService(cli, cfg.Service)
	return
}

// -i reverse
// trace

func main() {
	var err error
	var q Query
	var Loglevel string
	var ShowVersion bool
	var ConfigFile string
	var Profile string
	var Query bool
	flag.BoolVar(&ShowVersion, "version", false, "show version")
	flag.StringVar(&Loglevel, "loglevel", "", "log level")
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&Profile, "profile", "", "run profile")
	flag.BoolVar(&Query, "q", false, "force do query")
	flag.BoolVar(&drivers.Insecure, "insecure", false, "don't check cert in https")
	flag.IntVar(&drivers.Timeout, "timeout", 0, "query timeout, in ms.")
	q.Parse()
	flag.Parse()

	if ShowVersion {
		fmt.Printf("version: %s\n", Version)
		return
	}

	cfg := &Config{}
	if ConfigFile != "" {
		drivers.LoadJson(ConfigFile, cfg, false)
	}

	if Loglevel != "" {
		cfg.Loglevel = Loglevel
	}
	drivers.SetLogging(cfg.Logfile, cfg.Loglevel)

	var cli drivers.Client
	q.Prepare()
	if cfg.Client != nil {
		cli = cfg.CreateClient()
	} else {
		cli = q.CreateClient()
	}
	if cli == nil {
		panic("can't create client")
	}
	logger.Debugf("%+v", cli)

	switch {
	case cfg.Service != nil && !Query:
		if Profile != "" {
			go func() {
				logger.Infof("golang profile %s", Profile)
				logger.Infof("golang profile result: %s",
					http.ListenAndServe(Profile, nil))
			}()
		}

		var srv drivers.Server
		srv = cfg.CreateService(cli)
		err = srv.Serve()
		if err != nil {
			logger.Error(err.Error())
			return
		}

	default:
		q.QueryAll(cli)
	}

	return
}
