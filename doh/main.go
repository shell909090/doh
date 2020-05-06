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
	DefaultConfigs = "doh.json;~/.doh.json;/etc/doh.json"
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
)

type Config struct {
	Logfile  string
	Loglevel string
	Service  json.RawMessage
	Client   json.RawMessage
	Aliases  map[string]string
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

func main() {
	var err error
	var ConfigFile string
	var Loglevel string
	var Profile string
	var Query bool
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&Loglevel, "loglevel", "", "log level")
	flag.StringVar(&Profile, "profile", "", "run profile")
	flag.BoolVar(&Query, "q", false, "query")
	flag.BoolVar(&FmtShort, "short", false, "show short answer")
	flag.BoolVar(&FmtJson, "json", false, "show json answer")
	flag.StringVar(&Subnet, "subnet", "", "edns client subnet")
	flag.StringVar(&QType, "type", "A", "qtype")
	flag.StringVar(&Driver, "driver", "", "client driver")
	flag.StringVar(&URL, "url", "", "client url")
	flag.BoolVar(&drivers.Insecure, "insecure", false, "don't check cert in https")
	flag.Parse()

	cfg := &Config{}
	err = drivers.LoadJson(DefaultConfigs, cfg)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if ConfigFile != "" {
		err = drivers.LoadJson(ConfigFile, cfg)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	if Loglevel != "" {
		cfg.Loglevel = Loglevel
	}
	drivers.SetLogging(cfg.Logfile, cfg.Loglevel)

	drivers.Aliases = &cfg.Aliases

	cli, err := cfg.CreateClient()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	logger.Debugf("%+v", cli)

	switch {
	case Query:
		for _, dn := range flag.Args() {
			QueryDN(cli, dn)
		}

	case cfg.Service != nil:
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
