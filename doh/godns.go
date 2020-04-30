package main

import (
	"net"

	"github.com/miekg/dns"
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

func (cli *DnsClient) Exchange(quiz *dns.Msg) (ans *dns.Msg, err error) {
	ans, _, err = cli.cli.Exchange(quiz, cli.Server)
	return
}

type DnsServer struct {
	Net  string
	Addr string
	cli  Client
}

func NewDnsServer(cli Client, Net, Addr string) (srv *DnsServer) {
	srv = &DnsServer{
		Net:  Net,
		Addr: Addr,
		cli:  cli,
	}
	return
}

func (srv *DnsServer) ServeDNS(w dns.ResponseWriter, quiz *dns.Msg) {
	logger.Infof("dns server query: %s", quiz.Question[0].Name)
	ans, err := srv.cli.Exchange(quiz)
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
	logger.Infof("dns server start.")
	err = server.ListenAndServe()
	return
}

func appendEdns0Subnet(m *dns.Msg, addr net.IP) {
	newOpt := true
	var o *dns.OPT
	for _, v := range m.Extra {
		if v.Header().Rrtype == dns.TypeOPT {
			o = v.(*dns.OPT)
			newOpt = false
			break
		}
	}
	if o == nil {
		o = &dns.OPT{}
		o.Hdr.Name = "."
		o.Hdr.Rrtype = dns.TypeOPT
	}

	e := &dns.EDNS0_SUBNET{
		Code:        dns.EDNS0SUBNET,
		SourceScope: 0,
		Address:     addr,
	}
	if e.Address.To4() == nil {
		e.Family = 2 // IP6
		e.SourceNetmask = net.IPv6len * 8
	} else {
		e.Family = 1 // IP4
		e.SourceNetmask = net.IPv4len * 8
	}

	o.Option = append(o.Option, e)
	if newOpt {
		m.Extra = append(m.Extra, o)
	}
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

func (wrapper *DnsClientSubnetWrapper) Exchange(quiz *dns.Msg) (ans *dns.Msg, err error) {
	appendEdns0Subnet(quiz, wrapper.addr)
	ans, err = wrapper.cli.Exchange(quiz)
	return
}
