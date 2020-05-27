package drivers

import (
	"context"
	"errors"
	"math/rand"

	"github.com/miekg/dns"
)

var (
	ErrUndeterminedNext = errors.New("can't determine next hop")
	ROOTDNS             = []string{
		"198.41.0.4", "199.9.14.201", "192.33.4.12", "199.7.91.13", "192.203.230.10",
		"192.5.5.241", "192.112.36.4", "198.97.190.53", "192.36.148.17",
		"192.58.128.30", "193.0.14.129", "199.7.83.42", "202.12.27.33",
	}
)

type RecursiveClient struct {
	dnscli  *dns.Client
	root_ip string
}

func NewRecursiveClient() (cli *RecursiveClient) {
	cli = &RecursiveClient{
		dnscli:  &dns.Client{Net: "udp"},
		root_ip: ROOTDNS[rand.Intn(len(ROOTDNS))],
	}

	return
}

func (cli *RecursiveClient) Url() (u string) {
	return "(trace)"
}

func (cli *RecursiveClient) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	query := NewRecursionQuery(ctx, cli.dnscli, quiz)
	err = query.Procedure()
	if err != nil {
		return
	}
	ans = query.ans
	return
}

type RecursionQuery struct {
	origquiz  *dns.Msg
	quiz      *dns.Msg
	ans       *dns.Msg
	interm    *dns.Msg
	nextaddrs []string
	nexthosts []string
	current   string
	client    *dns.Client
	ctx       context.Context
	finished  bool
	deep      int
}

func NewRecursionQuery(ctx context.Context, client *dns.Client, quiz *dns.Msg) (query *RecursionQuery) {
	query = &RecursionQuery{
		origquiz:  quiz,
		quiz:      quiz.Copy(),
		ans:       &dns.Msg{},
		nextaddrs: ROOTDNS,
		client:    client,
		ctx:       ctx,
	}
	query.quiz.MsgHdr.RecursionDesired = false
	query.ans.SetReply(query.quiz)
	return
}

func (query *RecursionQuery) SelectServer() (err error) {
	if len(query.nextaddrs) != 0 {
		query.current = query.nextaddrs[rand.Intn(len(query.nextaddrs))]
		return
	}

	if len(query.nexthosts) == 0 {
		panic("can't determine next hop")
	}

	host := query.nexthosts[rand.Intn(len(query.nexthosts))]

	rquiz := &dns.Msg{}
	rquiz.SetQuestion(host, dns.TypeA)

	rquery := NewRecursionQuery(query.ctx, query.client, rquiz)
	rquery.deep = query.deep + 1
	err = rquery.Procedure()
	if err != nil {
		return
	}

	for _, rr := range rquery.ans.Answer {
		switch v := rr.(type) {
		case *dns.A:
			query.current = v.A.String()
			return
		}
	}
	return
}

func (query *RecursionQuery) Query() (err error) {
	for i := 0; i < 2; i++ {
		err = query.SelectServer()
		if err != nil {
			return
		}
		question := query.quiz.Question[0]
		logger.Infof("%d doh -[%s %s]-> %s",
			query.deep, question.Name, dns.TypeToString[question.Qtype], query.current)

		query.interm, _, err = query.client.ExchangeContext(query.ctx, query.quiz, query.current+":53")
		if err == nil {
			break
		}
		logger.Infof(err.Error())
	}

	switch {
	case err == nil:
	case query.interm != nil:
		query.ans.MsgHdr.Rcode = query.interm.MsgHdr.Rcode
		return
	default:
		query.ans.MsgHdr.Rcode = dns.RcodeServerFailure
		return
	}

	logger.Debug(query.interm.String())
	query.nextaddrs = nil
	query.nexthosts = nil
	return
}

func (query *RecursionQuery) ParseInterm() {
	for _, rr := range query.interm.Answer {
		switch rr.Header().Rrtype {
		case dns.TypeCNAME:
			v := rr.(*dns.CNAME)
			query.quiz.Question[0].Name = v.Target
			query.ans.Answer = append(query.ans.Answer, v)
			query.nextaddrs = ROOTDNS
			logger.Infof("%d doh <-[%s CNAME]- %s", query.deep, v.Target, query.current)
			query.deep++

		case query.quiz.Question[0].Qtype:
			query.ans.Answer = append(query.ans.Answer, rr)
			query.finished = true
		}
	}

	if query.finished {
		logger.Infof("%d doh <-[final]- %s", query.deep, query.current)
	}

	Name := ""
	Qtype := ""
	Authority := make(map[string]interface{}, 0)
	for _, rr := range query.interm.Ns {
		switch v := rr.(type) {
		case *dns.NS:
			Qtype = "NS"
			Name = v.Hdr.Name
			Authority[v.Ns] = nil
			query.nexthosts = append(query.nexthosts, v.Ns)

		case *dns.SOA:
			Qtype = "SOA"
			Name = v.Hdr.Name
			Authority[v.Ns] = nil
			query.nexthosts = append(query.nexthosts, v.Ns)
			break
		}
	}

	if len(query.nexthosts) != 0 {
		logger.Infof("%d doh <-[%s %s %d]- %s",
			query.deep, Name, Qtype, len(query.nexthosts), query.current)
		logger.Infof("%+v", query.nexthosts)
	}

	nexts := make([]string, 0)
	for _, rr := range query.interm.Extra {
		if v, ok := rr.(*dns.A); ok {
			// rrtree.DefaultTree.AddRecord(rr)
			if _, ok := Authority[v.Hdr.Name]; ok {
				nexts = append(nexts, v.A.String())
			}
		}
	}
	if len(nexts) != 0 {
		query.nextaddrs = nexts
		logger.Infof("%+v", query.nextaddrs)
	}

	return
}

func (query *RecursionQuery) Procedure() (err error) {
	for i := 0; i < 10 && !query.finished; i++ {
		err = query.Query()
		if err != nil {
			return
		}
		query.ParseInterm()
	}
	return
}
