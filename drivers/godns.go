package drivers

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/url"

	"github.com/miekg/dns"
)

var (
	ErrParseSubnet = errors.New("failed to parse subnet")
)

type DnsClient struct {
	host string
	URL  string
	cli  *dns.Client
}

func NewDnsClient(URL string) (cli *DnsClient, err error) {
	var u *url.URL
	u, err = url.Parse(URL)
	if err != nil {
		logger.Error(err.Error())
		return
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
	net              string
	addr             string
	clientAddr       net.IP
	clientMask       uint8
	cli              Client
}

func NewDnsServer(cli Client, URL string, body json.RawMessage) (srv *DnsServer, err error) {
	var u *url.URL
	u, err = url.Parse(URL)
	if err != nil {
		logger.Error(err.Error())
		return
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

	if srv.EdnsClientSubnet != "" {
		srv.clientAddr, srv.clientMask, err = ParseSubnet(srv.EdnsClientSubnet)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}
	return
}

func (srv *DnsServer) ServeDNS(w dns.ResponseWriter, quiz *dns.Msg) {
	logger.Infof("dns server query: %s", quiz.Question[0].Name)

	if srv.clientAddr != nil {
		AppendEdns0Subnet(quiz, srv.clientAddr, srv.clientMask)
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

func (srv *DnsServer) Run() (err error) {
	// FIXME: cert?
	server := &dns.Server{
		Net:     srv.net,
		Addr:    srv.addr,
		Handler: srv,
	}
	logger.Infof("dns server start. listen in %s://%s", srv.net, srv.addr)
	err = server.ListenAndServe()
	return
}
