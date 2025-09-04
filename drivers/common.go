package drivers

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/miekg/dns"
	logging "github.com/op/go-logging"
)

var (
	ErrConfigParse = errors.New("config parse error")
	ErrParseSubnet = errors.New("failed to parse subnet")
	ErrRequest     = errors.New("failed to get response")
	ErrBadQtype    = errors.New("wrong or unsupported qtype")
	logger         = logging.MustGetLogger("drivers")
	Insecure       bool
	Timeout        int
)

type Client interface {
	Url() (u string)
	Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error)
}

type Server interface {
	Serve() (err error)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func LoadJson(configfiles string, cfg interface{}, ignore_notexist bool) {
	for _, conf := range strings.Split(configfiles, ";") {
		if strings.HasPrefix(conf, "~/") {
			usr, _ := user.Current()
			conf = filepath.Join(usr.HomeDir, conf[2:])
		}

		file, err := os.Open(conf)
		if err != nil {
			if ignore_notexist {
				continue
			}
			panic(err.Error())
		}
		defer file.Close()

		dec := json.NewDecoder(file)
		err = dec.Decode(&cfg)
		if err != nil {
			panic(err.Error())
		}
	}

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

func GuessDriver(URL string) (driver string, err error) {
	var u *url.URL
	u, err = url.Parse(URL)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	switch u.Scheme {
	case "udp", "tcp", "tcp-tls":
		driver = "dns"

	case "http", "https":
		switch u.Path {
		case "/resolve":
			driver = "google"
		case "/dns-query":
			driver = "rfc8484"
		case "/d":
			driver = "dnspod"
		default:
			driver = "doh"
		}

	default:
		err = ErrConfigParse
	}

	return
}

func GuessPort(u *url.URL) {
	if strings.Contains(u.Host, ":") {
		return
	}

	switch u.Scheme {
	case "udp", "tcp":
		u.Host = net.JoinHostPort(u.Host, "53")
	case "tcp-tls":
		u.Host = net.JoinHostPort(u.Host, "853")
	default:
	}
	return
}

func AppendEdns0Subnet(m *dns.Msg, addr net.IP, mask uint8) {
	opt := m.IsEdns0()
	if opt == nil {
		opt = &dns.OPT{}
		opt.Hdr.Name = "."
		opt.Hdr.Rrtype = dns.TypeOPT
		m.Extra = append(m.Extra, opt)
	}

	e := &dns.EDNS0_SUBNET{
		Code:          dns.EDNS0SUBNET,
		SourceNetmask: mask,
		SourceScope:   0,
		Address:       addr,
	}
	if addr.To4() == nil {
		e.Family = 2 // IP6
	} else {
		e.Family = 1 // IP4
	}

	opt.Option = append(opt.Option, e)
}

func ParseSubnet(subnet string) (ip net.IP, mask uint8, err error) {
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		err = nil
		ipstring := strings.SplitN(subnet, "/", 2)[0]
		ip = net.ParseIP(ipstring)
		switch {
		case ip == nil:
			err = ErrParseSubnet
			return
		case ip.To4() == nil:
			mask = net.IPv6len * 8
		default:
			mask = net.IPv4len * 8
		}
		return
	}
	one, _ := ipnet.Mask.Size()
	mask = uint8(one)
	return
}

func HttpSetEdns0Subnet(w http.ResponseWriter, req *http.Request, ecs1, ecs2 string, quiz *dns.Msg) (err error) {
	var addr net.IP
	var mask uint8
	switch {
	case ecs1 != "":
		addr, mask, err = ParseSubnet(ecs1)
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		AppendEdns0Subnet(quiz, addr, mask)

	case ecs2 == "client":
		addr, mask, err = ParseSubnet(req.RemoteAddr)
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		AppendEdns0Subnet(quiz, addr, mask)

	case ecs2 != "":
		addr, mask, err = ParseSubnet(ecs2)
		if err != nil {
			panic(err.Error())
		}
		AppendEdns0Subnet(quiz, addr, mask)
	}
	return
}
