package drivers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/url"

	"github.com/miekg/dns"
)

type DnsClient struct {
	URL  string
	host string
	cli  *dns.Client
}

func NewDnsClient(URL string) (cli *DnsClient) {
	u, err := url.Parse(URL)
	if err != nil {
		panic(err.Error())
	}

	GuessPort(u)

	cli = &DnsClient{
		host: u.Host,
		URL:  URL,
		cli: &dns.Client{
			Net: u.Scheme,
		},
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
	EdnsClientSubnet string
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
	// FIXME: cert? https://godoc.org/github.com/miekg/dns#Server
	server := &dns.Server{
		Net:     srv.net,
		Addr:    srv.addr,
		Handler: srv,
	}
	logger.Infof("dns server start. listen in %s://%s", srv.net, srv.addr)
	err = server.ListenAndServe()
	return
}
