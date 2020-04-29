package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/miekg/dns"
	logging "github.com/op/go-logging"
)

type Client interface {
	Init() (err error)
	Exchange(quiz *dns.Msg) (ans *dns.Msg, err error)
}

type Server interface {
	Init() (err error)
	Run() (err error)
}

type Profile map[string]interface{}

type Config struct {
	Logfile        string
	Loglevel       string
	InputProtocol  string `json:"input-protocol"`
	InputURL       string `json:"input-url"`
	InputCertFile  string `json:"input-cert-file"`
	InputKeyFile   string `json:"input-key-file"`
	Input          Profile
	OutputProtocol string `json:"output-protocol"`
	OutputURL      string `json:"output-url"`
	Output         Profile
}

var (
	ErrConfigParse = errors.New("config parse error")
	logger         = logging.MustGetLogger("")
)

func LoadJson(configfile string, cfg interface{}) (err error) {
	file, err := os.Open(configfile)
	if err != nil {
		return
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	err = dec.Decode(&cfg)
	return
}

func SetLogging(logfile, loglevel string) (err error) {
	var file *os.File
	file = os.Stdout

	if loglevel == "" {
		loglevel = "WARNING"
	}
	if logfile != "" {
		file, err = os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			panic(err.Error())
		}
	}
	logging.SetBackend(logging.NewLogBackend(file, "", 0))
	logging.SetFormatter(logging.MustStringFormatter(
		"%{time:01-02 15:04:05.000}[%{level}] %{shortpkg}/%{shortfile}: %{message}"))
	lv, err := logging.LogLevel(loglevel)
	if err != nil {
		panic(err.Error())
	}
	logging.SetLevel(lv, "")
	return
}

func CreateOutput(cfg *Config) (client Client, err error) {
	switch cfg.OutputProtocol {
	case "dns", "":
		var u *url.URL
		u, err = url.Parse(cfg.OutputURL)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		client = &DnsClient{
			Net:    u.Scheme,
			Server: u.Host,
		}
	case "google":
		client = &GoogleClient{
			URL: cfg.OutputURL,
		}
	case "rfc8484":
		client = &Rfc8484Client{
			URL: cfg.OutputURL,
		}
	default:
		err = ErrConfigParse
		return
	}

	sprofile, err := json.Marshal(cfg.Output)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	err = json.Unmarshal(sprofile, client)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	err = client.Init()
	return
}

func CreateInput(cfg *Config, client Client) (srv Server, err error) {
	var u *url.URL
	u, err = url.Parse(cfg.InputURL)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	switch cfg.InputProtocol {
	case "dns", "":
		srv = &DnsServer{
			Net:    u.Scheme,
			Addr:   u.Host,
			Client: client,
		}
	case "doh":
		srv = &DoHServer{
			Scheme:   u.Scheme,
			Addr:     u.Host,
			CertFile: cfg.InputCertFile,
			KeyFile:  cfg.InputKeyFile,
			Client:   client,
		}
	default:
		err = ErrConfigParse
		return
	}

	err = srv.Init()
	return
}

func QueryDN(client Client, dn string) (err error) {
	quiz := &dns.Msg{}
	quiz.SetQuestion(dns.Fqdn(dn), dns.TypeA)
	ans, err := client.Exchange(quiz)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	fmt.Println(ans.String())
	return
}

func main() {
	var ConfigFile string
	var Loglevel string
	var Logfile string
	var GoProfile string
	var Query bool
	var Serve string
	var Listen string
	var CertFile string
	var KeyFile string
	var Protocol string
	var URL string
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&Logfile, "logfile", "", "log file")
	flag.StringVar(&Loglevel, "loglevel", "", "log level")
	flag.StringVar(&GoProfile, "profile", "", "run profile")
	flag.BoolVar(&Query, "query", false, "query")
	flag.StringVar(&Serve, "serve", "", "input protocol")
	flag.StringVar(&Listen, "listen", "", "input listen address")
	flag.StringVar(&CertFile, "cert", "", "input cert file")
	flag.StringVar(&KeyFile, "key", "", "input key file")
	flag.StringVar(&Protocol, "protocol", "", "output protocol")
	flag.StringVar(&URL, "url", "", "output url")
	flag.Parse()

	cfg := &Config{}
	if ConfigFile != "" {
		err := LoadJson(ConfigFile, cfg)
		if err != nil {
			log.Fatal(err)
			return
		}
	}

	if Logfile != "" {
		cfg.Logfile = Logfile
	}
	if Loglevel != "" {
		cfg.Loglevel = Loglevel
	}
	SetLogging(cfg.Logfile, cfg.Loglevel)

	if Serve != "" {
		cfg.InputProtocol = Serve
	}
	if Listen != "" {
		cfg.InputURL = Listen
	}
	if Protocol != "" {
		cfg.OutputProtocol = Protocol
	}
	if URL != "" {
		cfg.OutputURL = URL
	}

	if GoProfile != "" {
		go func() {
			logger.Infof("golang profile %s", GoProfile)
			logger.Infof("golang profile result: %s",
				http.ListenAndServe(GoProfile, nil))
		}()
	}

	logger.Debugf("config: %+v", cfg)
	client, err := CreateOutput(cfg)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	logger.Debugf("client: %+v", client)

	if Query {
		logger.Debugf("domains: %+v", flag.Args())
		for _, dn := range flag.Args() {
			QueryDN(client, dn)
		}
	} else {
		var srv Server
		srv, err = CreateInput(cfg, client)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		err = srv.Run()
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}
}
