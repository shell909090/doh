package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/shell909090/doh/drivers"
)

type Query struct {
	AliasesFile      string
	Aliases          map[string]string
	ResolvFile       string
	Driver           string
	URL              string
	URLs             []string
	TCP              bool
	Tries            int
	Subnet           string
	QType            string
	QClass           string
	NoRecurse        bool
	CheckingDisabled bool
	FmtQuestion      bool
	FmtShort         bool
	FmtJson          bool
	Microseconds     bool
	Trace            bool
	DNlist           []string
}

func (q *Query) Parse() {
	flag.StringVar(&q.AliasesFile, "alias", "", "aliases file")
	flag.StringVar(&q.ResolvFile, "resolv", "/etc/resolv.conf", "file path of resolv.conf")
	flag.StringVar(&q.Driver, "driver", "", "client driver")
	flag.StringVar(&q.URL, "s", "", "server url to query")
	flag.BoolVar(&q.TCP, "tcp", false, "use tcp as default protocol")
	flag.IntVar(&q.Tries, "tries", 0, "retries")
	flag.StringVar(&q.Subnet, "subnet", "", "edns client subnet")
	flag.StringVar(&q.QType, "t", "A", "resource record type to query")
	flag.StringVar(&q.QClass, "c", "IN", "resource record class to query")
	flag.BoolVar(&q.NoRecurse, "norecurse", false, "not desire recurse")
	flag.BoolVar(&q.CheckingDisabled, "cd", false, "checking disabled, don't check ENSSEC validation")
	flag.BoolVar(&q.FmtQuestion, "question", false, "print question")
	flag.BoolVar(&q.FmtShort, "short", false, "show short answer")
	flag.BoolVar(&q.FmtJson, "json", false, "show json answer")
	flag.BoolVar(&q.Microseconds, "u", false, "print query times in microseconds instead of milliseconds")
	flag.BoolVar(&q.Trace, "trace", false, "trace the query")
}

func (q *Query) Prepare() {
	drivers.LoadJson(DEFAULT_ALIASES, &q.Aliases, true)
	if q.AliasesFile != "" {
		drivers.LoadJson(q.AliasesFile, &q.Aliases, false)
	}

	if q.URL != "" {
		q.URLs = append(q.URLs, q.FillURL(q.URL))
	}

	for _, p := range flag.Args() {
		_, typeok := dns.StringToType[p]
		_, classok := dns.StringToClass[p]
		switch {
		case strings.HasPrefix(p, "@"):
			q.URLs = append(q.URLs, q.FillURL(p[1:]))
		case typeok:
			q.QType = p
		case classok:
			q.QClass = p
		default:
			q.DNlist = append(q.DNlist, p)
		}
	}

	if q.ResolvFile != "" {
		cfg, err := dns.ClientConfigFromFile(q.ResolvFile)
		if err != nil {
			panic("no server and can't read resolv.conf")
		}
		// don't append, user can't overwrite resolv if it's append.
		if len(q.URLs) == 0 {
			for _, srv := range cfg.Servers {
				q.URLs = append(q.URLs, q.FillURL(srv))
			}
		}
		if drivers.Timeout == 0 {
			drivers.Timeout = cfg.Timeout * 1000
		}
		if q.Tries == 0 {
			q.Tries = cfg.Attempts
		}
	}

	logger.Debugf("%+v", q)
	return
}

func (q *Query) FillURL(u string) (URL string) {
	URL = u
	if AliasURL, ok := q.Aliases[URL]; ok {
		URL = AliasURL
	}
	if !strings.Contains(URL, "://") {
		if q.TCP {
			URL = "tcp://" + URL
		} else {
			URL = "udp://" + URL
		}
	}
	return
}

func (q *Query) CreateClient() (cli drivers.Client) {
	var header *drivers.DriverHeader

	if len(q.URLs) == 0 {
		return
	}

	switch {
	case q.Trace:
		cli = drivers.NewRecursiveClient()

	case q.Tries <= 1:
		header = &drivers.DriverHeader{
			Driver: q.Driver,
			URL:    q.URLs[0],
		}
		cli = header.CreateClient(nil)

	default:
		reties := &drivers.RetiesClient{Tries: q.Tries}
		for _, URL := range q.URLs {
			header = &drivers.DriverHeader{
				Driver: q.Driver,
				URL:    URL,
			}
			reties.AddClient(header.CreateClient(nil))
		}
		cli = reties
	}

	return
}

func (q *Query) NewQuiz(dn string) (quiz *dns.Msg) {
	qtype, ok := dns.StringToType[q.QType]
	if !ok {
		panic(ErrParameter.Error())
	}

	quiz = &dns.Msg{}
	quiz.SetQuestion(dns.Fqdn(dn), qtype)
	quiz.SetEdns0(4096, true)
	if q.NoRecurse {
		quiz.MsgHdr.RecursionDesired = false
	}
	if q.CheckingDisabled {
		quiz.MsgHdr.CheckingDisabled = true
	}

	if q.QClass != "" {
		qclass, ok := dns.StringToClass[q.QClass]
		if !ok {
			panic(ErrParameter.Error())
		}
		quiz.Question[0].Qclass = qclass
	}

	if q.Subnet != "" {
		addr, mask, err := drivers.ParseSubnet(q.Subnet)
		if err != nil {
			panic(err.Error())
		}
		drivers.AppendEdns0Subnet(quiz, addr, mask)
	}
	return
}

func (q *Query) QueryDN(cli drivers.Client, quiz *dns.Msg) (err error) {
	ctx := context.Background()
	start := time.Now()

	ans, err := cli.Exchange(ctx, quiz)
	if err != nil {
		return
	}

	elapsed := time.Since(start)

	switch {
	case q.FmtShort:
		q.PrintShort(ans)

	case q.FmtJson:
		q.PrintJson(quiz, ans)

	default:
		if q.FmtQuestion {
			fmt.Println(quiz.String())
		}
		fmt.Println(ans.String())
		if q.Microseconds {
			fmt.Printf(";; Query time: %d usec\n", elapsed.Microseconds())
		} else {
			fmt.Printf(";; Query time: %d msec\n", elapsed.Milliseconds())
		}
		fmt.Printf(";; SERVER: %s\n", cli.Url())
		fmt.Printf(";; WHEN: %s\n\n", start.Format(time.UnixDate))
	}

	return
}

func (q *Query) PrintShort(ans *dns.Msg) {
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
	return
}

func (q *Query) PrintJson(quiz, ans *dns.Msg) {
	jsonresp := &drivers.DNSMsg{}
	err := jsonresp.FromAnswer(quiz, ans)
	if err != nil {
		panic(err.Error())
	}

	var bresp []byte
	bresp, err = json.Marshal(jsonresp)
	if err != nil {
		panic(err.Error())
	}

	fmt.Printf("%s", string(bresp))
	return
}

func (q *Query) QueryAll(cli drivers.Client) {
	for _, dn := range q.DNlist {
		quiz := q.NewQuiz(dn)
		err := q.QueryDN(cli, quiz)
		if err != nil {
			logger.Error(err.Error())
		}
	}
	return
}
