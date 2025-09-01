package drivers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/miekg/dns"
)

type DnsClient struct {
	URL     string
	Timeout int
	host    string
	cli     *dns.Client
}

func NewDnsClient(URL string, body json.RawMessage) (cli *DnsClient) {
	cli = &DnsClient{}
	if body != nil {
		err := json.Unmarshal(body, &cli)
		if err != nil {
			panic(err.Error())
		}
	}
	cli.URL = URL

	if Timeout != 0 {
		cli.Timeout = Timeout
	}

	u, err := url.Parse(URL)
	if err != nil {
		panic(err.Error())
	}
	GuessPort(u)

	cli.host = u.Host
	cli.cli = &dns.Client{
		Net: u.Scheme,
	}

	if cli.Timeout != 0 {
		cli.cli.Timeout = time.Duration(cli.Timeout) * time.Millisecond
	}

	return
}

func (cli *DnsClient) Url() (u string) {
	return cli.URL
}

func (cli *DnsClient) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	ans, _, err = cli.cli.ExchangeContext(ctx, quiz, cli.host)
	return
}

type DnsServer struct {
	EdnsClientSubnet string `json:"edns-client-subnet"`
	CertFile         string
	CertKeyFile      string
	net              string
	addr             string
	cert             *tls.Certificate
	cli              Client
}

func NewDnsServer(cli Client, URL string, body json.RawMessage) (srv *DnsServer) {
	u, err := url.Parse(URL)
	if err != nil {
		panic(err.Error())
	}

	GuessPort(u)

	srv = &DnsServer{
		net:  u.Scheme,
		addr: u.Host,
		cli:  cli,
	}

	if body != nil {
		err = json.Unmarshal(body, &srv)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	if srv.CertFile != "" && srv.CertKeyFile != "" {
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(srv.CertFile, srv.CertKeyFile)
		if err != nil {
			panic(err.Error)
		}
		srv.cert = &cert
	}

	return
}

func (srv *DnsServer) ServeDNS(w dns.ResponseWriter, quiz *dns.Msg) {
	logger.Infof("dns server query: %s", quiz.Question[0].Name)

	var addr net.IP
	var mask uint8
	var err error
	switch srv.EdnsClientSubnet {
	case "":
	case "client":
		raddr := w.RemoteAddr()
		switch taddr := raddr.(type) {
		case *net.TCPAddr:
			addr = taddr.IP
		case *net.UDPAddr:
			addr = taddr.IP
		default:
			panic(fmt.Sprintf("unknown addr %s", raddr.Network()))
		}
		mask = 32
		AppendEdns0Subnet(quiz, addr, mask)

	default:
		addr, mask, err = ParseSubnet(srv.EdnsClientSubnet)
		if err != nil {
			panic(err.Error())
		}
		AppendEdns0Subnet(quiz, addr, mask)
	}

	ctx := context.Background()
	ans, err := srv.cli.Exchange(ctx, quiz)
	if err != nil {
		// FIXME: google dns not 2xx or 3xx
		logger.Error(err.Error())
		return
	}
	if ans == nil {
		logger.Error("response is nil.")
		return
	}

	err = w.WriteMsg(ans)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	return
}

func (srv *DnsServer) Serve() (err error) {
	server := &dns.Server{
		Net:     srv.net,
		Addr:    srv.addr,
		Handler: srv,
	}
	if srv.cert != nil {
		server.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{*srv.cert},
		}
	}

	logger.Infof("dns server start. listen in %s://%s", srv.net, srv.addr)
	err = server.ListenAndServe()
	return
}
