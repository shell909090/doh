package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
	logging "github.com/op/go-logging"
	"github.com/shell909090/doh/drivers"
)

const (
	DEFAULT_CONFIGS = "doh.json;~/.doh.json;/etc/doh.json"
	DEFAULT_ALIASES = "doh-aliases.json;~/.doh-aliases.json"
)

var (
	ErrConfigParse        = errors.New("config parse error")
	ErrParameter          = errors.New("parameter error")
	logger                = logging.MustGetLogger("")
	Version        string = "unknown"
	Driver         string
	URL            string
	FmtShort       bool
	FmtJson        bool
	Microseconds   bool
)

type Config struct {
	Logfile  string
	Loglevel string
	Service  json.RawMessage
	Client   json.RawMessage
}

func (cfg *Config) CreateClient() (cli drivers.Client) {
	var header drivers.DriverHeader
	if cfg.Client != nil {
		err := json.Unmarshal(cfg.Client, &header)
		if err != nil {
			panic(err.Error())
		}
	}

	if URL != "" {
		header.URL = URL
	}
	if Driver != "" {
		header.Driver = Driver
	}

	// TODO: if it's totally empty, driver, url. read from resolve.conf

	cli = header.CreateClient(cfg.Client)

	return
}

func (cfg *Config) CreateService(cli drivers.Client) (srv drivers.Server) {
	var err error
	var header drivers.DriverHeader
	if cfg.Service != nil {
		err = json.Unmarshal(cfg.Service, &header)
		if err != nil {
			panic(err.Error())
		}
	}

	srv = header.CreateService(cli, cfg.Service)
	return
}

func NewQuiz(dn, QType, QClass, Subnet string) (quiz *dns.Msg) {
	var err error

	qtype, ok := dns.StringToType[QType]
	if !ok {
		err = ErrParameter
		panic(err.Error())
	}

	quiz = &dns.Msg{}
	quiz.SetQuestion(dns.Fqdn(dn), qtype)

	if QClass != "" {
		qclass, ok := dns.StringToClass[QClass]
		if !ok {
			err = ErrParameter
			panic(err.Error())
		}
		quiz.Question[0].Qclass = qclass
	}

	if Subnet != "" {
		var addr net.IP
		var mask uint8
		addr, mask, err = drivers.ParseSubnet(Subnet)
		if err != nil {
			panic(err.Error())
		}
		drivers.AppendEdns0Subnet(quiz, addr, mask)
	}
	return
}

func QueryDN(cli drivers.Client, quiz *dns.Msg) (err error) {
	ctx := context.Background()
	start := time.Now()

	ans, err := cli.Exchange(ctx, quiz)
	if err != nil {
		return
	}

	elapsed := time.Since(start)

	switch {
	case FmtShort:
		for _, rr := range ans.Answer {
			switch v := rr.(type) {
			case *dns.A:
				fmt.Println(v.A.String())
			case *dns.AAAA:
				fmt.Println(v.AAAA.String())
			case *dns.CNAME:
				fmt.Println(v.Target)
			}
		}

	case FmtJson:
		jsonresp := &drivers.DNSMsg{}
		err = jsonresp.FromAnswer(quiz, ans)
		if err != nil {
			return
		}

		var bresp []byte
		bresp, err = json.Marshal(jsonresp)
		if err != nil {
			return
		}

		fmt.Printf("%s", string(bresp))

	default:
		fmt.Println(ans.String())
		if Microseconds {
			fmt.Printf(";; Query time: %d usec\n", elapsed.Microseconds())
		} else {
			fmt.Printf(";; Query time: %d msec\n", elapsed.Milliseconds())
		}
		fmt.Printf(";; SERVER: %s\n", cli.Url())
		fmt.Printf(";; WHEN: %s\n\n", start.Format(time.UnixDate))
	}

	return
}

// -f batch mode
// -i reverse
// trace
// tries
// norecurse

func main() {
	var err error
	var ShowVersion bool
	var ConfigFile string
	var AliasesFile string
	var Profile string
	var Query bool
	var Subnet string
	var QType string
	var QClass string
	flag.BoolVar(&ShowVersion, "version", false, "show version")
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&AliasesFile, "alias", "", "aliases file")
	flag.StringVar(&Profile, "profile", "", "run profile")
	flag.StringVar(&Driver, "driver", "", "client driver")
	flag.StringVar(&URL, "s", "", "server url to query")
	flag.StringVar(&Subnet, "subnet", "", "edns client subnet")
	flag.BoolVar(&drivers.Insecure, "insecure", false, "don't check cert in https")
	flag.IntVar(&drivers.Timeout, "timeout", 0, "query timeout")
	flag.BoolVar(&Query, "q", false, "force do query")
	flag.StringVar(&QType, "t", "A", "resource record type to query.")
	flag.StringVar(&QClass, "c", "", "resource record class to query.")
	flag.BoolVar(&FmtShort, "short", false, "show short answer")
	flag.BoolVar(&FmtJson, "json", false, "show json answer")
	flag.BoolVar(&Microseconds, "u", false, "print query times in microseconds instead of milliseconds")
	flag.Parse()

	if ShowVersion {
		fmt.Printf("version: %s\n", Version)
		return
	}

	cfg := &Config{}
	drivers.LoadJson(DEFAULT_ALIASES, cfg, true)
	if ConfigFile != "" {
		drivers.LoadJson(ConfigFile, cfg, false)
	}
	drivers.SetLogging(cfg.Logfile, cfg.Loglevel)

	var dnlist []string
	for _, p := range flag.Args() {
		_, typeok := dns.StringToType[p]
		_, classok := dns.StringToClass[p]
		switch {
		case strings.HasPrefix(p, "@"):
			if URL == "" {
				URL = p[1:]
			}
		case typeok:
			if QType == "" {
				QType = p
			}
		case classok:
			if QClass == "" {
				QClass = p
			}
		default:
			dnlist = append(dnlist, p)
		}
	}

	var Aliases map[string]string
	drivers.LoadJson(DEFAULT_ALIASES, &Aliases, true)
	if AliasesFile != "" {
		drivers.LoadJson(AliasesFile, &Aliases, false)
	}
	if AliasURL, ok := Aliases[URL]; ok {
		URL = AliasURL
	}

	if !strings.Contains(URL, "://") {
		URL = "udp://" + URL
	}

	cli := cfg.CreateClient()
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
		for _, dn := range dnlist {
			quiz := NewQuiz(dn, QType, QClass, Subnet)
			err = QueryDN(cli, quiz)
			if err != nil {
				logger.Error(err.Error())
			}
		}
	}

	return
}
