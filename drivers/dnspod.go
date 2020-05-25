package drivers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

type DnsPodClient struct {
	URL       string
	transport *http.Transport
}

func NewDnsPodClient(URL string, body json.RawMessage) (cli *DnsPodClient, err error) {
	cli = &DnsPodClient{}
	if body != nil {
		err = json.Unmarshal(body, &cli)
		if err != nil {
			logger.Error(err.Error())
			return
		}
	}

	cli.URL = URL
	cli.transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	return
}

func (cli *DnsPodClient) Url() (u string) {
	return cli.URL
}

func (cli *DnsPodClient) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	if quiz.Question[0].Qtype != dns.TypeA {
		return nil, ErrBadQtype
	}

	req, err := http.NewRequestWithContext(ctx, "GET", cli.URL, nil)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	query := req.URL.Query()
	query.Add("dn", quiz.Question[0].Name)
	query.Add("ttl", "1")

	opt := quiz.IsEdns0()
	if opt != nil {
		for _, o := range opt.Option {
			if e, ok := o.(*dns.EDNS0_SUBNET); ok {
				query.Add("ip", e.Address.String())
				break
			}
		}
	}

	req.URL.RawQuery = query.Encode()

	resp, err := cli.transport.RoundTrip(req)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = ErrRequest
		return
	}

	bresp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err.Error())
		return
	}
	sresp := string(bresp)

	ans = &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:       quiz.Id,
			Response: true,
			Opcode:   dns.OpcodeQuery,
			Rcode:    dns.RcodeSuccess,
		},
		Compress: quiz.Compress,
		Question: quiz.Question,
	}

	for _, rec := range strings.Split(sresp, ";") {
		var n int = 0
		if strings.Contains(rec, ",") {
			r := strings.SplitN(rec, ",", 2)
			n, err = strconv.Atoi(r[1])
			if err != nil {
				logger.Error(err.Error())
				return
			}
			rec = r[0]
		}
		rr := &dns.A{
			Hdr: dns.RR_Header{
				Name:   quiz.Question[0].Name,
				Rrtype: dns.TypeA,
				Class:  quiz.Question[0].Qclass,
				Ttl:    uint32(n),
			},
			A: net.ParseIP(rec),
		}
		ans.Answer = append(ans.Answer, rr)
	}

	if opt != nil {
		ans.Extra = append(ans.Extra, opt)
	}

	return
}
