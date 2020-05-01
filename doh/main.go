package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/miekg/dns"
	logging "github.com/op/go-logging"
)

type Client interface {
	Url() (u string)
	Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error)
}

type Server interface {
	Run() (err error)
}

type Config struct {
	Logfile          string
	Loglevel         string
	InputProtocol    string `json:"input-protocol"`
	InputURL         string `json:"input-url"`
	InputCertFile    string `json:"input-cert-file"`
	InputKeyFile     string `json:"input-key-file"`
	EdnsClientSubnet string `json:"edns-client-subnet"`
	OutputProtocol   string `json:"output-protocol"`
	OutputURL        string `json:"output-url"`
	OutputInsecure   bool   `json:"output-insecure"`
}

var (
	ErrConfigParse = errors.New("config parse error")
	logger         = logging.MustGetLogger("")
	Short          bool
	Subnet         string
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
	var u *url.URL
	u, err = url.Parse(cfg.OutputURL)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	if cfg.OutputProtocol == "" {
		switch u.Scheme {
		case "udp", "tcp", "tcp-tls":
			cfg.OutputProtocol = "dns"

		case "http", "https":
			switch u.Path {
			case "/resolve":
				cfg.OutputProtocol = "google"
			case "/dns-query":
				cfg.OutputProtocol = "rfc8484"
			default:
				err = ErrConfigParse
				return
			}

		default:
			err = ErrConfigParse
			return
		}
	}

	switch cfg.OutputProtocol {
	case "dns":
		client = NewDnsClient(cfg.OutputURL, u)
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

	if cfg.InputProtocol == "" {
		switch u.Scheme {
		case "udp", "tcp", "tcp-tls":
			cfg.InputProtocol = "dns"
		case "http", "https":
			cfg.InputProtocol = "doh"
		default:
			err = ErrConfigParse
			return
		}
	}

	switch cfg.InputProtocol {
	case "dns":
		srv, err = NewDnsServer(cli, u.Scheme, u.Host, cfg.EdnsClientSubnet)
	case "doh":
		srv, err = NewDoHServer(cli, u.Scheme, u.Host, cfg.InputCertFile, cfg.InputKeyFile, cfg.EdnsClientSubnet)
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
	var ConfigFile string
	var Loglevel string
	var Profile string
	var Query bool
	var IPv4 bool
	var IPv6 bool
	var Protocol string
	var URL string
	var Insecure bool
	flag.StringVar(&ConfigFile, "config", "", "config file")
	flag.StringVar(&Loglevel, "loglevel", "", "log level")
	flag.StringVar(&Profile, "profile", "", "run profile")
	flag.BoolVar(&Query, "q", false, "query")
	flag.BoolVar(&Short, "short", false, "show short answer")
	flag.StringVar(&Subnet, "subnet", "", "query subnet")
	flag.BoolVar(&IPv4, "4", false, "query ipv4 only")
	flag.BoolVar(&IPv6, "6", false, "query ipv6 only")
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

	case cfg.InputURL != "":
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
