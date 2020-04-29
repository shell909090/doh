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

type Exchanger interface {
	Init() (err error)
	Exchange(quiz *dns.Msg) (ans *dns.Msg, err error)
}

type Profile map[string]interface{}

type Config struct {
	Logfile    string
	Loglevel   string
	InputType  string `json:"input-type"`
	InputURL   string `json:"input-url"`
	Input      Profile
	OutputType string `json:"output-type"`
	OutputURL  string `json:"output-url"`
	Output     Profile
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

func CreateOutput(cfg *Config) (xchg Exchanger, err error) {
	switch cfg.OutputType {
	case "dns", "":
		var u *url.URL
		u, err = url.Parse(cfg.OutputURL)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		xchg = &DnsClient{
			Net:    u.Scheme,
			Server: u.Host,
		}
	case "google":
		xchg = &GoogleClient{
			URL: cfg.OutputURL,
		}
	case "rfc8484":
		xchg = &Rfc8484Client{
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
	err = json.Unmarshal(sprofile, xchg)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	err = xchg.Init()
	return
}

func QueryDN(xchg Exchanger, dn string) (err error) {
	quiz := &dns.Msg{}
	quiz.SetQuestion(dns.Fqdn(dn), dns.TypeA)
	ans, err := xchg.Exchange(quiz)
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
	var GoProfile string
	var Type string
	var URL string
	var Query bool
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&Loglevel, "loglevel", "", "log level")
	flag.StringVar(&GoProfile, "profile", "", "run profile")
	flag.BoolVar(&Query, "query", true, "query")
	flag.StringVar(&Type, "type", "", "output type")
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

	if Loglevel != "" {
		cfg.Loglevel = Loglevel
	}
	SetLogging(cfg.Logfile, cfg.Loglevel)

	if Type != "" {
		cfg.OutputType = Type
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
	xchg, err := CreateOutput(cfg)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	logger.Debugf("exchanger: %+v", xchg)

	if Query {
		logger.Debugf("domains: %+v", flag.Args())
		for _, dn := range flag.Args() {
			QueryDN(xchg, dn)
		}
	}

}
