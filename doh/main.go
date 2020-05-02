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
)

const (
	DefaultConfigs = "doh.json;~/.doh.json;/etc/doh.json"
)

var (
	ErrConfigParse = errors.New("config parse error")
	Short          bool
	Subnet         string
	Driver         string
	URL            string
	Insecure       bool
	Aliases        *map[string]string
)

type ClientHeader struct {
	Driver string
	URL    string
}

func (header *ClientHeader) CreateClient(body json.RawMessage) (cli Client, err error) {
	if URL, ok := (*Aliases)[header.URL]; ok {
		header.URL = URL
	}

	if header.Driver == "" {
		header.Driver, err = GuessDriver(header.URL)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	switch header.Driver {
	case "dns":
		cli, err = NewDnsClient(header.URL)
	case "google":
		cli, err = NewGoogleClient(header.URL, body)
	case "rfc8484":
		cli, err = NewRfc8484Client(header.URL, body)
	default:
		err = ErrConfigParse
	}

	return
}

type Config struct {
	Logfile          string
	Loglevel         string
	ServiceDriver    string `json:"service-driver"`
	ServiceURL       string `json:"service-url"`
	CertFile         string `json:"cert-file"`
	KeyFile          string `json:"key-file"`
	EdnsClientSubnet string `json:"edns-client-subnet"`
	Client           json.RawMessage
	Aliases          map[string]string
}

func (cfg *Config) CreateOutput() (cli Client, err error) {
	var header ClientHeader
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

func (cfg *Config) CreateInput(cli Client) (srv Server, err error) {
	if cfg.ServiceDriver == "" {
		cfg.ServiceDriver, err = GuessDriver(cfg.ServiceURL)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	switch cfg.ServiceDriver {
	case "dns":
		srv, err = NewDnsServer(cli, cfg.ServiceURL, cfg.EdnsClientSubnet)
	case "doh", "http", "https":
		srv, err = NewDoHServer(cli, cfg.ServiceURL, cfg.CertFile, cfg.KeyFile, cfg.EdnsClientSubnet)
	default:
		err = ErrConfigParse
		return
	}

	if err != nil {
		logger.Error(err.Error())
		return
	}
	return
}

func QueryDN(cli Client, dn string, qtype uint16) (err error) {
	ctx := context.Background()
	quiz := &dns.Msg{}
	quiz.SetQuestion(dns.Fqdn(dn), qtype)

	if Subnet != "" {
		var addr net.IP
		var mask uint8
		addr, mask, err = ParseSubnet(Subnet)
		if err != nil {
			logger.Error(err.Error())
			return
		}
		appendEdns0Subnet(quiz, addr, mask)
	}

	start := time.Now()

	ans, err := cli.Exchange(ctx, quiz)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	elapsed := time.Since(start)

	if Short {
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
	} else {
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
	var IP string
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&Loglevel, "loglevel", "", "log level")
	flag.StringVar(&Profile, "profile", "", "run profile")
	flag.BoolVar(&Query, "q", false, "query")
	flag.BoolVar(&Short, "short", false, "show short answer")
	flag.StringVar(&Subnet, "subnet", "", "edns client subnet")
	flag.StringVar(&IP, "ip", "4", "ip version to query")
	flag.StringVar(&Driver, "driver", "", "client driver")
	flag.StringVar(&URL, "url", "", "client url")
	flag.BoolVar(&Insecure, "insecure", false, "don't check cert in https")
	flag.Parse()

	cfg := &Config{}
	err = LoadJson(DefaultConfigs, cfg)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if ConfigFile != "" {
		err = LoadJson(ConfigFile, cfg)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	if Loglevel != "" {
		cfg.Loglevel = Loglevel
	}
	SetLogging(cfg.Logfile, cfg.Loglevel)

	Aliases = &cfg.Aliases

	cli, err := cfg.CreateOutput()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	logger.Debugf("%+v", cli)

	switch {
	case Query:
		for _, dn := range flag.Args() {
			if strings.Contains(IP, "4") {
				QueryDN(cli, dn, dns.TypeA)
			}
			if strings.Contains(IP, "6") {
				QueryDN(cli, dn, dns.TypeAAAA)
			}
		}

	case cfg.ServiceURL != "":
		if Profile != "" {
			go func() {
				logger.Infof("golang profile %s", Profile)
				logger.Infof("golang profile result: %s",
					http.ListenAndServe(Profile, nil))
			}()
		}

		var srv Server
		srv, err = cfg.CreateInput(cli)
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
