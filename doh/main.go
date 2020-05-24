package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
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
	ErrConfigParse = errors.New("config parse error")
	ErrParameter   = errors.New("parameter error")
	logger         = logging.MustGetLogger("")
	FmtShort       bool
	FmtJson        bool
	QType          string
	Subnet         string
	Driver         string
	URL            string
	Version        string = "unknown"
)

type Config struct {
	Logfile  string
	Loglevel string
	Service  json.RawMessage
	Client   json.RawMessage
}

func (cfg *Config) CreateClient() (cli drivers.Client, err error) {
	var header drivers.DriverHeader
	if cfg.Client != nil {
		err = json.Unmarshal(cfg.Client, &header)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	if URL != "" {
		header.URL = URL
	}
	if Driver != "" {
		header.Driver = Driver
	}

	cli, err = header.CreateClient(cfg.Client)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	return
}

func (cfg *Config) CreateService(cli drivers.Client) (srv drivers.Server, err error) {
	var header drivers.DriverHeader
	if cfg.Service != nil {
		err = json.Unmarshal(cfg.Service, &header)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	srv, err = header.CreateService(cli, cfg.Service)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	return
}

func QueryDN(cli drivers.Client, dn string) (err error) {
	qtype, ok := dns.StringToType[QType]
	if !ok {
		err = ErrParameter
		return
	}

	ctx := context.Background()
	quiz := &dns.Msg{}
	quiz.SetQuestion(dns.Fqdn(dn), qtype)

	if Subnet != "" {
		var addr net.IP
		var mask uint8
		addr, mask, err = drivers.ParseSubnet(Subnet)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		drivers.AppendEdns0Subnet(quiz, addr, mask)
	}

	start := time.Now()

	ans, err := cli.Exchange(ctx, quiz)
	if err != nil {
		logger.Error(err.Error())
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
			logger.Error(err.Error())
			return
		}

		var bresp []byte
		bresp, err = json.Marshal(jsonresp)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		fmt.Printf("%s", string(bresp))

	default:
		fmt.Println(ans.String())
		fmt.Printf(";; Query time: %d msec\n", elapsed.Milliseconds())
		fmt.Printf(";; SERVER: %s\n", cli.Url())
		fmt.Printf(";; WHEN: %s\n\n", start.Format(time.UnixDate))
	}

	return
}

// -c class
// -f batch mode
// -i reverse
// -u print query times in microseconds instead of milliseconds.
// trace

func main() {
	var err error
	var ConfigFile string
	var Profile string
	var AliasesFile string
	var ShowVersion bool
	var Query bool
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&AliasesFile, "alias", "", "aliases file")
	flag.StringVar(&Profile, "profile", "", "run profile")
	flag.BoolVar(&FmtShort, "short", false, "show short answer")
	flag.BoolVar(&FmtJson, "json", false, "show json answer")
	flag.StringVar(&Subnet, "subnet", "", "edns client subnet")
	flag.BoolVar(&Query, "q", false, "force do query")
	flag.StringVar(&QType, "t", "A", "resource record type to query.")
	flag.StringVar(&Driver, "driver", "", "client driver")
	flag.StringVar(&URL, "s", "", "server url to query")
	flag.BoolVar(&drivers.Insecure, "insecure", false, "don't check cert in https")
	flag.BoolVar(&ShowVersion, "version", false, "show version")
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

	var Aliases map[string]string
	drivers.LoadJson(DEFAULT_ALIASES, &Aliases, true)
	if AliasesFile != "" {
		drivers.LoadJson(AliasesFile, &Aliases, false)
	}
	if AliasURL, ok := Aliases[URL]; ok {
		URL = AliasURL
	}

	cli, err := cfg.CreateClient()
	if err != nil {
		logger.Error(err.Error())
		return
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
		srv, err = cfg.CreateService(cli)
		if err != nil {
			logger.Error(err.Error())
			return
		}

		err = srv.Serve()
		if err != nil {
			logger.Error(err.Error())
			return
		}

	default:
		for _, dn := range flag.Args() {
			QueryDN(cli, dn)
		}
	}

	return
}
