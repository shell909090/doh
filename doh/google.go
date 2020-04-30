package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/miekg/dns"
)

func ParseUint(s string) (n uint64) {
	n, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		logger.Error("ParseUint error.")
		return
	}
	return
}

type GoogleClient struct {
	URL       string
	Insecure  bool
	transport *http.Transport
}

func NewGoogleClient(URL string, Insecure bool) (cli *GoogleClient) {
	cli = &GoogleClient{
		URL:      URL,
		Insecure: Insecure,
		transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	if cli.Insecure {
		cli.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return
}

func (cli *GoogleClient) Exchange(quiz *dns.Msg) (ans *dns.Msg, err error) {
	req, err := http.NewRequest("GET", cli.URL, nil)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	query := req.URL.Query()
	query.Add("name", quiz.Question[0].Name)
	query.Add("type", fmt.Sprintf("%v", quiz.Question[0].Qtype))

	opt := quiz.IsEdns0()
	if opt != nil {
		for _, o := range opt.Option {
			if e, ok := o.(*dns.EDNS0_SUBNET); ok {
				subnet := e.Address.String()
				query.Add("edns_client_subnet", subnet)
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

	jsonresp := &DNSMsg{}
	err = json.NewDecoder(resp.Body).Decode(&jsonresp)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	ans, err = jsonresp.TranslateAnswer(quiz)
	if err != nil {
		return
	}
	return
}

type DNSMsg struct {
	Status             int32         `json:"Status,omitempty"`
	TC                 bool          `json:"TC,omitempty"`
	RD                 bool          `json:"RD,omitempty"`
	RA                 bool          `json:"RA,omitempty"`
	AD                 bool          `json:"AD,omitempty"`
	CD                 bool          `json:"CD,omitempty"`
	Question           []DNSQuestion `json:"Question,omitempty"`
	Answer             []DNSRR       `json:"Answer,omitempty"`
	Authority          []DNSRR       `json:"Authority,omitempty"`
	Additional         []DNSRR       `json:"Additional,omitempty"`
	Edns_client_subnet string        `json:"edns_client_subnet,omitempty"`
	Comment            string        `json:"Comment,omitempty"`
}

type DNSQuestion struct {
	Name string `json:"name,omitempty"`
	Type int32  `json:"type,omitempty"`
}

type DNSRR struct {
	Name string `json:"name,omitempty"`
	Type int32  `json:"type,omitempty"`
	TTL  int32  `json:"TTL,omitempty"`
	Data string `json:"data,omitempty"`
}

func (msg *DNSMsg) TranslateAnswer(quiz *dns.Msg) (ans *dns.Msg, err error) {
	ans = &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:                 quiz.Id,
			Response:           (msg.Status == 0),
			Opcode:             dns.OpcodeQuery,
			Authoritative:      false,
			Truncated:          msg.TC,
			RecursionDesired:   msg.RD,
			RecursionAvailable: msg.RA,
			AuthenticatedData:  msg.AD,
			CheckingDisabled:   msg.CD,
			Rcode:              int(msg.Status),
		},
		Compress: quiz.Compress,
	}

	for idx, q := range msg.Question {
		ans.Question = append(ans.Question,
			dns.Question{
				q.Name,
				uint16(q.Type),
				quiz.Question[idx].Qclass,
			})
	}

	TranslateRRs(&msg.Answer, &ans.Answer)
	TranslateRRs(&msg.Authority, &ans.Ns)
	TranslateRRs(&msg.Additional, &ans.Extra)

	// TODO: is this right?
	clientip := strings.Split(msg.Edns_client_subnet, "/")[0]
	addr := net.ParseIP(clientip)
	if addr != nil {
		appendEdns0Subnet(ans, addr)
	}

	return
}

func TranslateRRs(jrs *[]DNSRR, rrs *[]dns.RR) {
	for _, jr := range *jrs {
		rr := jr.Translate()
		if rr != nil {
			*rrs = append(*rrs, rr)
		}
	}
}

func (jr *DNSRR) Translate() (rr dns.RR) {
	switch uint16(jr.Type) {
	case dns.TypeA:
		rr = &dns.A{
			A: net.ParseIP(jr.Data),
		}
	case dns.TypeNS:
		rr = &dns.NS{
			Ns: jr.Data,
		}
	case dns.TypeMD:
		rr = &dns.MD{
			Md: jr.Data,
		}
	case dns.TypeMF:
		rr = &dns.MF{
			Mf: jr.Data,
		}
	case dns.TypeCNAME:
		rr = &dns.CNAME{
			Target: jr.Data,
		}
	case dns.TypeSOA:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 7 {
			return
		}
		rr = &dns.SOA{
			Ns:      parts[0],
			Mbox:    parts[1],
			Serial:  uint32(ParseUint(parts[2])),
			Refresh: uint32(ParseUint(parts[3])),
			Retry:   uint32(ParseUint(parts[4])),
			Expire:  uint32(ParseUint(parts[5])),
			Minttl:  uint32(ParseUint(parts[6])),
		}
	case dns.TypeMB:
		rr = &dns.MB{
			Mb: jr.Data,
		}
	case dns.TypeMG:
		rr = &dns.MG{
			Mg: jr.Data,
		}
	case dns.TypeMR:
		rr = &dns.MR{
			Mr: jr.Data,
		}
	case dns.TypeNULL:
	case dns.TypePTR:
		rr = &dns.PTR{
			Ptr: jr.Data,
		}
	case dns.TypeHINFO:
	case dns.TypeMINFO:
	case dns.TypeMX:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 2 {
			return
		}
		rr = &dns.MX{
			Preference: uint16(ParseUint(parts[0])),
			Mx:         parts[1],
		}
	case dns.TypeTXT:
		rr = &dns.TXT{
			Txt: strings.Split(jr.Data, " "),
		}
	case dns.TypeRP:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 2 {
			return
		}
		rr = &dns.RP{
			Mbox: parts[0],
			Txt:  parts[1],
		}
	case dns.TypeAAAA:
		rr = &dns.AAAA{
			AAAA: net.ParseIP(jr.Data),
		}
	case dns.TypeSRV:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 4 {
			return
		}
		rr = &dns.SRV{
			Priority: uint16(ParseUint(parts[0])),
			Weight:   uint16(ParseUint(parts[1])),
			Port:     uint16(ParseUint(parts[2])),
			Target:   parts[3],
		}
	case dns.TypeSPF:
		rr = &dns.SPF{
			Txt: strings.Split(jr.Data, " "),
		}
	case dns.TypeDS:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 4 {
			return
		}
		rr = &dns.DS{
			KeyTag:     uint16(ParseUint(parts[0])),
			Algorithm:  uint8(ParseUint(parts[1])),
			DigestType: uint8(ParseUint(parts[2])),
			Digest:     parts[3],
		}
	case dns.TypeSSHFP:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 3 {
			return
		}
		rr = &dns.SSHFP{
			Algorithm:   uint8(ParseUint(parts[0])),
			Type:        uint8(ParseUint(parts[1])),
			FingerPrint: parts[2],
		}
	case dns.TypeRRSIG:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 9 {
			return
		}
		rrsig := &dns.RRSIG{
			Algorithm:  uint8(ParseUint(parts[1])),
			Labels:     uint8(ParseUint(parts[2])),
			OrigTtl:    uint32(ParseUint(parts[3])),
			Expiration: uint32(ParseUint(parts[4])),
			Inception:  uint32(ParseUint(parts[5])),
			KeyTag:     uint16(ParseUint(parts[6])),
			SignerName: parts[7],
			Signature:  parts[8],
		}
		var ok bool
		if rrsig.TypeCovered, ok = dns.StringToType[strings.ToUpper(parts[0])]; !ok {
			return
		}
		rr = rrsig
	case dns.TypeNSEC:
		nsec := &dns.NSEC{}
		parts := strings.Split(jr.Data, " ")
		nsec.NextDomain = parts[0]
		for _, d := range parts[1:] {
			if typeBit, ok := dns.StringToType[strings.ToUpper(d)]; ok {
				nsec.TypeBitMap = append(nsec.TypeBitMap, typeBit)
			}
		}
		rr = nsec
	case dns.TypeDNSKEY:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 4 {
			return
		}
		rr = &dns.DNSKEY{
			Flags:     uint16(ParseUint(parts[0])),
			Protocol:  uint8(ParseUint(parts[1])),
			Algorithm: uint8(ParseUint(parts[2])),
			PublicKey: parts[3],
		}
	case dns.TypeNSEC3:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 7 {
			return
		}
		nsec3 := &dns.NSEC3{
			Hash:       uint8(ParseUint(parts[0])),
			Flags:      uint8(ParseUint(parts[1])),
			Iterations: uint16(ParseUint(parts[2])),
			SaltLength: uint8(ParseUint(parts[3])),
			Salt:       parts[4],
			HashLength: uint8(ParseUint(parts[5])),
			NextDomain: parts[6],
		}
		for _, d := range parts[7:] {
			if t, ok := dns.StringToType[strings.ToUpper(d)]; ok {
				nsec3.TypeBitMap = append(nsec3.TypeBitMap, t)
			}
		}
		rr = nsec3
	case dns.TypeNSEC3PARAM:
		parts := strings.Split(jr.Data, " ")
		if len(parts) < 5 {
			return
		}
		rr = &dns.NSEC3PARAM{
			Hash:       uint8(ParseUint(parts[0])),
			Flags:      uint8(ParseUint(parts[1])),
			Iterations: uint16(ParseUint(parts[2])),
			SaltLength: uint8(ParseUint(parts[3])),
			Salt:       parts[4],
		}
	}
	hdr := &dns.RR_Header{
		Name:     jr.Name,
		Rrtype:   uint16(jr.Type),
		Ttl:      uint32(jr.TTL),
		Class:    dns.ClassINET,
		Rdlength: uint16(len(jr.Data)),
	}
	*(rr.Header()) = *hdr
	return
}
