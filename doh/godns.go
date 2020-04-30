package main

import (
	"context"
	"errors"
	"net"

	"github.com/miekg/dns"
)

var (
	ErrParseSubnet = errors.New("failed to parse subnet")
)

type DnsClient struct {
	Net    string
	Server string
	cli    *dns.Client
}

func NewDnsClient(Net, Server string) (cli *DnsClient) {
	cli = &DnsClient{
		Net:    Net,
		Server: Server,
		cli: &dns.Client{
			Net: Net,
		},
	}
	return
}

func (cli *DnsClient) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	ans, _, err = cli.cli.ExchangeContext(ctx, quiz, cli.Server)
	return
}

type DnsServer struct {
	Net              string
	Addr             string
	EdnsClientSubnet string
	clientAddr       net.IP
	clientMask       uint8
	cli              Client
}

func NewDnsServer(cli Client, Net, Addr, EdnsClientSubnet string) (srv *DnsServer, err error) {
	srv = &DnsServer{
		Net:              Net,
		Addr:             Addr,
		EdnsClientSubnet: EdnsClientSubnet,
		cli:              cli,
	}
	if EdnsClientSubnet != "" {
		srv.clientAddr, srv.clientMask, err = ParseSubnet(EdnsClientSubnet)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}
	return
}

func (srv *DnsServer) ServeDNS(w dns.ResponseWriter, quiz *dns.Msg) {
	logger.Infof("dns server query: %s", quiz.Question[0].Name)
	if srv.EdnsClientSubnet != "" {
		appendEdns0Subnet(quiz, srv.clientAddr, srv.clientMask)
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
	server := &dns.Server{
		Net:     srv.Net,
		Addr:    srv.Addr,
		Handler: srv,
	}
	logger.Infof("dns server start. listen in %s://%s", srv.Net, srv.Addr)
	err = server.ListenAndServe()
	return
}

func ParseSubnet(subnet string) (ip net.IP, mask uint8, err error) {
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		ip = net.ParseIP(subnet)
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

func appendEdns0Subnet(m *dns.Msg, addr net.IP, mask uint8) {
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

type DnsClientSubnetWrapper struct {
	ClientSubnet string
	addr         net.IP
	cli          Client
}

func NewDnsClientSubnetWrapper(cli Client, clientSubnet string) (wrapper *DnsClientSubnetWrapper) {
	wrapper = &DnsClientSubnetWrapper{
		ClientSubnet: clientSubnet,
		addr:         net.ParseIP(clientSubnet),
		cli:          cli,
	}
	return

}

func (wrapper *DnsClientSubnetWrapper) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	appendEdns0Subnet(quiz, wrapper.addr, 32) // TODO:
	ans, err = wrapper.cli.Exchange(ctx, quiz)
	return
}
