package drivers

import (
	"context"
	"errors"
	"math/rand"
	"reflect"
	"strings"

	"github.com/miekg/dns"
)

var (
	ErrUndeterminedNext = errors.New("can't determine next hop")
)

type RecursiveClient struct {
	client      *dns.Client
	Cache       map[string][]string
	NameServers map[string][]string
}

func NewRecursiveClient() (cli *RecursiveClient) {
	cli = &RecursiveClient{
		client: &dns.Client{Net: "udp"},
		Cache: map[string][]string{
			"a.root-servers.net": []string{"198.41.0.4"},
			"b.root-servers.net": []string{"199.9.14.201"},
			"c.root-servers.net": []string{"192.33.4.12"},
			"d.root-servers.net": []string{"199.7.91.13"},
			"e.root-servers.net": []string{"192.203.230.10"},
			"f.root-servers.net": []string{"192.5.5.241"},
			"g.root-servers.net": []string{"192.112.36.4"},
			"h.root-servers.net": []string{"198.97.190.53"},
			"i.root-servers.net": []string{"192.36.148.17"},
			"j.root-servers.net": []string{"192.58.128.30"},
			"k.root-servers.net": []string{"193.0.14.129"},
			"l.root-servers.net": []string{"199.7.83.42"},
			"m.root-servers.net": []string{"202.12.27.33"},
		},
		NameServers: map[string][]string{
			".": []string{
				"a.root-servers.net", "b.root-servers.net", "c.root-servers.net", "d.root-servers.net",
				"e.root-servers.net", "f.root-servers.net", "g.root-servers.net", "h.root-servers.net",
				"i.root-servers.net", "j.root-servers.net", "k.root-servers.net", "l.root-servers.net",
				"m.root-servers.net",
			},
		},
	}

	return
}

func (client *RecursiveClient) ReadSection(section []dns.RR) (err error) {
	for _, rr := range section {
		if v, ok := rr.(*dns.A); ok {
			client.Cache[v.Hdr.Name] = append(client.Cache[v.Hdr.Name], v.A.String())
		}
	}
	return
}

func (client *RecursiveClient) AddNameServer(name, server string) {
	client.NameServers[name] = append(client.NameServers[name], server)
	return
}

func (client *RecursiveClient) SetNameServer(name, server string) {
	client.NameServers[name] = nil
	client.NameServers[name] = append(client.NameServers[name], server)
	return
}

func (client *RecursiveClient) MatchNameServers(domain string) (suffix string, servers []string) {
	length := 0
	for name, srv := range client.NameServers {
		if !strings.HasSuffix(domain, name) {
			continue
		}
		if len(name) > length {
			suffix = name
			servers = srv
			length = len(name)
		}
	}
	return
}

func (cli *RecursiveClient) Url() (u string) {
	return "(trace)"
}

func (cli *RecursiveClient) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	query := NewRecursiveQuery(ctx, cli, quiz)
	err = query.Procedure()
	if err != nil {
		return
	}
	ans = query.ans
	return
}

type RecursiveQuery struct {
	client  *RecursiveClient
	ctx     context.Context
	deep    int
	quiz    *dns.Msg
	ans     *dns.Msg
	current string
}

func NewRecursiveQuery(ctx context.Context, client *RecursiveClient, quiz *dns.Msg) (query *RecursiveQuery) {
	query = &RecursiveQuery{
		client: client,
		ctx:    ctx,
		quiz:   quiz.Copy(),
		ans:    &dns.Msg{},
	}
	query.quiz.MsgHdr.RecursionDesired = false
	query.quiz.SetEdns0(4096, true)
	query.ans.SetReply(query.quiz)
	return
}

func (query *RecursiveQuery) SelectServer() (host, addr string, err error) {
	question := query.quiz.Question[0]
	suffix, servers := query.client.MatchNameServers(question.Name)
	logger.Infof("%d %s match %s in ns cache.", query.deep, question.Name, suffix)

	host = servers[rand.Intn(len(servers))]

	addrs, ok := query.client.Cache[host]
	if ok {
		addr = addrs[rand.Intn(len(addrs))]
		return
	}

	logger.Infof("%d query A record for host %s", query.deep, host)

	quiz := &dns.Msg{}
	quiz.SetQuestion(host, dns.TypeA)
	quiz.SetEdns0(4096, true)

	rquery := NewRecursiveQuery(query.ctx, query.client, quiz)
	rquery.deep = query.deep + 1
	err = rquery.Procedure()
	if err != nil {
		return
	}

	query.client.ReadSection(rquery.ans.Answer)
	addrs, ok = query.client.Cache[host]
	if ok {
		addr = addrs[rand.Intn(len(addrs))]
		logger.Infof("%d %s: %s", query.deep, host, addr)
		return
	}

	err = ErrUndeterminedNext
	return
}

func (query *RecursiveQuery) Query() (interm *dns.Msg, err error) {
	var addr string
	for i := 0; i < 3; i++ {
		query.current, addr, err = query.SelectServer()
		if err != nil {
			return
		}

		question := query.quiz.Question[0]
		logger.Infof("%d doh -[%s %s]-> %s|%s",
			query.deep, question.Name, dns.TypeToString[question.Qtype], query.current, addr)

		interm, _, err = query.client.client.ExchangeContext(query.ctx, query.quiz, addr+":53")
		if err == nil {
			break
		}
		logger.Infof(err.Error())
	}

	switch {
	case err == nil:
	case interm != nil:
		query.ans.MsgHdr.Rcode = interm.MsgHdr.Rcode
		return
	default:
		query.ans.MsgHdr.Rcode = dns.RcodeServerFailure
		return
	}

	logger.Debug(interm.String())
	return
}

func (query *RecursiveQuery) ParseAnswer(section []dns.RR) (finished bool) {
	for _, rr := range section {
		switch rr.Header().Rrtype {
		case dns.TypeCNAME:
			v := rr.(*dns.CNAME)
			query.quiz.Question[0].Name = v.Target
			query.ans.Answer = append(query.ans.Answer, v)
			logger.Infof("%d doh <-[%s CNAME]- %s", query.deep, v.Target, query.current)
			query.deep++

		case query.quiz.Question[0].Qtype:
			query.ans.Answer = append(query.ans.Answer, rr)
			finished = true
		}
	}
	return
}

func (query *RecursiveQuery) ParseAuthority(section []dns.RR) {
	domain := query.quiz.Question[0].Name
	name := ""
	qtype := ""
	count := 0
	for _, rr := range section {
		switch v := rr.(type) {
		case *dns.NS:
			qtype = "NS"
			name = v.Hdr.Name
			query.client.AddNameServer(name, v.Ns)

			if !strings.HasSuffix(domain, name) {
				count++
			}

		case *dns.SOA:
			qtype = "SOA"
			name = v.Hdr.Name
			query.client.SetNameServer(name, v.Ns)

			if strings.HasSuffix(domain, name) {
				count++
			}
			break
		}
	}

	if count != 0 {
		logger.Infof("%d doh <-[%s %s %d]- %s",
			query.deep, name, qtype, count, query.current)
		logger.Infof("%d ns: %v",
			query.deep, reflect.ValueOf(query.client.NameServers).MapKeys())
	}

	return
}

func (query *RecursiveQuery) Procedure() (err error) {
	var interm *dns.Msg
	for i := 0; i < 10; i++ {
		interm, err = query.Query()
		if err != nil {
			return
		}
		if query.ParseAnswer(interm.Answer) {
			logger.Infof("%d doh <-[final]- %s", query.deep, query.current)
			break
		}
		query.ParseAuthority(interm.Ns)
		query.client.ReadSection(interm.Extra)
	}
	return
}
