package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/miekg/dns"
	logging "github.com/op/go-logging"
)

type Client interface {
	Exchange(quiz *dns.Msg) (ans *dns.Msg, err error)
}

type Server interface {
	Run() (err error)
}

type Config struct {
	Logfile        string
	Loglevel       string
	InputProtocol  string `json:"input-protocol"`
	InputURL       string `json:"input-url"`
	InputCertFile  string `json:"input-cert-file"`
	InputKeyFile   string `json:"input-key-file"`
	OutputProtocol string `json:"output-protocol"`
	OutputURL      string `json:"output-url"`
	OutputInsecure bool   `json:"output-insecure"`
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
		client = NewDnsClient(u.Scheme, u.Host)
	case "google":
		client = NewGoogleClient(cfg.OutputURL, cfg.OutputInsecure)
	case "rfc8484":
		client = NewRfc8484Client(cfg.OutputURL, cfg.OutputInsecure)
	default:
		err = ErrConfigParse
		return
	}
	return
}

func CreateInput(cfg *Config, cli Client) (srv Server, err error) {
	var u *url.URL
	u, err = url.Parse(cfg.InputURL)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	switch cfg.InputProtocol {
	case "dns", "":
		srv = NewDnsServer(cli, u.Scheme, u.Host)
	case "doh":
		srv = NewDoHServer(cli, u.Scheme, u.Host, cfg.InputCertFile, cfg.InputKeyFile)
	default:
		err = ErrConfigParse
		return
	}

	return
}

func QueryDN(cli Client, dn string) (err error) {
	quiz := &dns.Msg{}
	quiz.SetQuestion(dns.Fqdn(dn), dns.TypeA)
	ans, err := cli.Exchange(quiz)
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
	var Profile string
	var Query bool
	var Protocol string
	var URL string
	var Insecure bool
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&Loglevel, "loglevel", "", "log level")
	flag.StringVar(&Profile, "profile", "", "run profile")
	flag.BoolVar(&Query, "q", false, "query")
	flag.StringVar(&Protocol, "protocol", "", "output protocol")
	flag.StringVar(&URL, "url", "", "output url")
	flag.BoolVar(&Insecure, "insecure", false, "output insecure")
	flag.Parse()

	cfg := &Config{}
	if ConfigFile != "" {
		LoadJson(ConfigFile, cfg)
	}

	if Loglevel != "" {
		cfg.Loglevel = Loglevel
	}
	SetLogging(cfg.Logfile, cfg.Loglevel)

	if Protocol != "" {
		cfg.OutputProtocol = Protocol
	}
	if URL != "" {
		cfg.OutputURL = URL
	}
	if Insecure {
		cfg.OutputInsecure = Insecure
	}

	logger.Debugf("config: %+v", cfg)
	cli, err := CreateOutput(cfg)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	switch {
	case Query:
		for _, dn := range flag.Args() {
			QueryDN(cli, dn)
		}

	case cfg.InputProtocol != "" && cfg.InputURL != "":
		if Profile != "" {
			go func() {
				logger.Infof("golang profile %s", Profile)
				logger.Infof("golang profile result: %s",
					http.ListenAndServe(Profile, nil))
			}()
		}

		var srv Server
		srv, err = CreateInput(cfg, cli)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		err = srv.Run()
		if err != nil {
			logger.Error(err.Error())
			return
		}

	default:
		logger.Error("no query nor server, quit.")
		return
	}
	return
}
