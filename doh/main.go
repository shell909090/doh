package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/miekg/dns"
	logging "github.com/op/go-logging"
)

type Exchanger interface {
	Exchange(quiz *dns.Msg) (ans *dns.Msg, rtt time.Duration, err error)
}

type Profile map[string]interface{}

type Config struct {
	Logfile  string
	Loglevel string
	Input    string
	Output   Profile
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
		loglevel = "INFO"
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

func CreateOutput(profile Profile) (xchg Exchanger, err error) {
	v, ok := profile["type"]
	if !ok {
		err = ErrConfigParse
		return
	}
	vs, ok := v.(string)
	if !ok {
		err = ErrConfigParse
		return
	}
	sprofile, err := json.Marshal(profile)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	switch vs {
	case "dns":
		profdns := &DnsProfile{}
		err = json.Unmarshal(sprofile, &profdns)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		logger.Debugf("prof dns: %+v", profdns)
		profdns.Init()
		xchg = profdns
	case "doh":
	default:
	}
	return
}

func QueryDN(xchg Exchanger, dn string) (err error) {
	quiz := &dns.Msg{}
	quiz.SetQuestion(dns.Fqdn(dn), dns.TypeA)
	ans, _, err := xchg.Exchange(quiz)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	fmt.Println(ans.String())
	return
}

func main() {
	var ConfigFile string
	var GoProfile string
	var Query bool
	flag.StringVar(&ConfigFile, "config", "doh.json", "config file")
	flag.StringVar(&GoProfile, "go profile", "", "run profile")
	flag.BoolVar(&Query, "query", true, "query")
	flag.Parse()

	cfg := &Config{}
	err := LoadJson(ConfigFile, cfg)
	if err != nil {
		log.Fatal(err)
		return
	}
	SetLogging(cfg.Logfile, cfg.Loglevel)

	if GoProfile != "" {
		go func() {
			logger.Infof("golang profile %s", GoProfile)
			logger.Infof("golang profile result: %s",
				http.ListenAndServe(GoProfile, nil))
		}()
	}

	logger.Debugf("config: %+v", cfg)
	xchg, err := CreateOutput(cfg.Output)

	if Query {
		logger.Debugf("domains: %+v", flag.Args())
		for _, dn := range flag.Args() {
			QueryDN(xchg, dn)
		}
	}

}
