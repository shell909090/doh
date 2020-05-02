package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
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

type ClientConfig struct {
	Driver   string          `json:"driver"`
	URL      string          `json:"url"`
	Insecure bool            `json:"insecure"`
	Subs     []*ClientConfig `json:"subs"`
}

func (cfg *ClientConfig) CreateClient() (cli Client, err error) {
	if URL, ok := (*Aliases)[cfg.URL]; ok {
		cfg.URL = URL
	}

	if cfg.Driver == "" {
		cfg.Driver, err = GuessDriver(cfg.URL)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	switch cfg.Driver {
	case "dns":
		cli, err = NewDnsClient(cfg.URL)
	case "google":
		cli = NewGoogleClient(cfg.URL, cfg.Insecure)
	case "rfc8484":
		cli = NewRfc8484Client(cfg.URL, cfg.Insecure)
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
	Client           *ClientConfig
	Aliases          map[string]string

	// OutputProtocol string `json:"output-protocol"`
	// OutputURL      string `json:"output-url"`
	// OutputInsecure bool   `json:"output-insecure"`
}

func (cfg *Config) CreateOutput() (cli Client, err error) {
	if URL != "" {
		if cfg.Client == nil {
			cfg.Client = &ClientConfig{}
		}
		cfg.Client.URL = URL
	}

	if cfg.Client == nil {
		err = ErrConfigParse
		logger.Error(err.Error())
		return
	}

	if Driver != "" {
		cfg.Client.Driver = Driver
	}

	if Insecure {
		cfg.Client.Insecure = Insecure
	}

	cli, err = cfg.Client.CreateClient()
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
	var IPv4 bool
	var IPv6 bool
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&Loglevel, "loglevel", "", "log level")
	flag.StringVar(&Profile, "profile", "", "run profile")
	flag.BoolVar(&Query, "q", false, "query")
	flag.BoolVar(&Short, "short", false, "show short answer")
	flag.StringVar(&Subnet, "subnet", "", "query subnet")
	flag.BoolVar(&IPv4, "4", false, "query ipv4 only")
	flag.BoolVar(&IPv6, "6", false, "query ipv6 only")
	flag.StringVar(&Driver, "driver", "", "output driver")
	flag.StringVar(&URL, "url", "", "output url")
	flag.BoolVar(&Insecure, "insecure", false, "output insecure")
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

	switch {
	case Query:
		for _, dn := range flag.Args() {
			switch {
			case !IPv4 && !IPv6:
				QueryDN(cli, dn, dns.TypeA)
				QueryDN(cli, dn, dns.TypeAAAA)
			case IPv4 && !IPv6:
				QueryDN(cli, dn, dns.TypeA)
			case !IPv4 && IPv6:
				QueryDN(cli, dn, dns.TypeAAAA)
			default:
				logger.Error("don't use -4 and -6 at the same time")
				return
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
