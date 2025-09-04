package drivers

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

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
	Timeout   int
	transport *http.Transport
}

func NewGoogleClient(URL string, body json.RawMessage) (cli *GoogleClient) {
	cli = &GoogleClient{}
	if body != nil {
		err := json.Unmarshal(body, &cli)
		if err != nil {
			panic(err.Error())
		}
	}

	cli.URL = URL
	if Insecure {
		cli.Insecure = Insecure
	}
	if Timeout != 0 {
		cli.Timeout = Timeout
	}

	cli.transport = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}
	if cli.Insecure {
		cli.transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	return
}

func (cli *GoogleClient) Url() (u string) {
	return cli.URL
}

func (cli *GoogleClient) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	if cli.Timeout != 0 {
		ctx, _ = context.WithTimeout(ctx, time.Duration(cli.Timeout)*time.Millisecond)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", cli.URL, nil)
	if err != nil {
		return
	}

	query := req.URL.Query()
	query.Add("name", quiz.Question[0].Name)
	query.Add("type", dns.TypeToString[quiz.Question[0].Qtype])

	opt := quiz.IsEdns0()
	if opt != nil {
		for _, o := range opt.Option {
			if e, ok := o.(*dns.EDNS0_SUBNET); ok {
				subnet := fmt.Sprintf("%s/%d", e.Address.String(), e.SourceNetmask)
				query.Add("edns_client_subnet", subnet)
				break
			}
		}
	}

	req.URL.RawQuery = query.Encode()
	logger.Debugf("query: %s", req.URL.RawQuery)

	resp, err := cli.transport.RoundTrip(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err = ErrRequest
		return
	}

	jsonresp := &DNSMsg{}
	err = json.NewDecoder(resp.Body).Decode(&jsonresp)
	if err != nil {
		return
	}

	ans, err = jsonresp.TranslateAnswer(quiz)
	if err != nil {
		return
	}

	return
}

type GoogleHandler struct {
	EdnsClientSubnet string
	cli              Client
}

func NewGoogleHandler(cli Client, EdnsClientSubnet string) (handler *GoogleHandler) {
	handler = &GoogleHandler{
		EdnsClientSubnet: EdnsClientSubnet,
		cli:              cli,
	}
	return
}

func (handler *GoogleHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	err := req.ParseForm()
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	quiz := &dns.Msg{}

	name := req.Form.Get("name")
	stype := req.Form.Get("type")
	qtype, ok := dns.StringToType[stype]
	if !ok {
		qtype = dns.TypeA
	}
	quiz.SetQuestion(dns.Fqdn(name), qtype)

	ecs := req.Form.Get("edns_client_subnet")
	err = HttpSetEdns0Subnet(w, req, ecs, handler.EdnsClientSubnet, quiz)
	if err != nil {
		return
	}

	if req.Form.Get("do") != "" {
		quiz.SetEdns0(4096, true)
	}

	logger.Infof("google server query: %s", quiz.Question[0].Name)

	ctx := context.Background()
	ans, err := handler.cli.Exchange(ctx, quiz)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	var bresp []byte
	if req.Form.Get("ct") == "application/dns-message" {
		bresp, err = ans.Pack()
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(http.StatusBadGateway)
			return
		}

	} else {
		jsonresp := &DNSMsg{}
		err = jsonresp.FromAnswer(quiz, ans)
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(http.StatusBadGateway)
			return
		}

		bresp, err = json.Marshal(jsonresp)
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Add("Content-Type", "application/dns-message")
	w.Header().Add("Cache-Control", "no-cache, max-age=0")
	w.WriteHeader(http.StatusOK)

	err = WriteFull(w, bresp)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	return
}

type DNSMsg struct {
	Status             int32         `json:"Status"`
	TC                 bool          `json:"TC"`
	RD                 bool          `json:"RD"`
	RA                 bool          `json:"RA"`
	AD                 bool          `json:"AD"`
	CD                 bool          `json:"CD"`
	Question           []DNSQuestion `json:"Question"`
	Answer             []DNSRR       `json:"Answer"`
	Authority          []DNSRR       `json:"Authority"`
	Additional         []DNSRR       `json:"Additional"`
	Edns_client_subnet string        `json:"edns_client_subnet,omitempty"`
	Comment            string        `json:"Comment,omitempty"`
}

type DNSQuestion struct {
	Name string `json:"name"`
	Type int32  `json:"type"`
}

type DNSRR struct {
	Name string `json:"name"`
	Type int32  `json:"type"`
	TTL  int32  `json:"TTL"`
	Data string `json:"data"`
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

	err = TranslateRRs(&msg.Answer, &ans.Answer)
	if err != nil {
		return
	}

	err = TranslateRRs(&msg.Authority, &ans.Ns)
	if err != nil {
		return
	}

	err = TranslateRRs(&msg.Additional, &ans.Extra)
	if err != nil {
		return
	}

	if msg.Edns_client_subnet != "" {
		var addr net.IP
		var mask uint8
		addr, mask, err = ParseSubnet(msg.Edns_client_subnet)
		if err != nil {
			logger.Errorf("can't parse subnet %s", msg.Edns_client_subnet)
			return
		}
		AppendEdns0Subnet(ans, addr, mask)
	}

	return
}

func TranslateRRs(jrs *[]DNSRR, rrs *[]dns.RR) (err error) {
	var rr dns.RR
	for _, jr := range *jrs {
		rr, err = jr.Translate()
		if err != nil {
			return
		}
		if rr != nil {
			*rrs = append(*rrs, rr)
		}
	}
	return
}

func (msg *DNSMsg) FromAnswer(quiz, ans *dns.Msg) (err error) {
	msg.Status = int32(ans.MsgHdr.Rcode)
	msg.TC = ans.MsgHdr.Truncated
	msg.RD = ans.MsgHdr.RecursionDesired
	msg.RA = ans.MsgHdr.RecursionAvailable
	msg.AD = ans.MsgHdr.AuthenticatedData
	msg.CD = ans.MsgHdr.CheckingDisabled

	for _, q := range ans.Question {
		msg.Question = append(msg.Question,
			DNSQuestion{
				Name: q.Name,
				Type: int32(q.Qtype),
			})
	}

	FromRRs(&msg.Answer, &ans.Answer)
	FromRRs(&msg.Authority, &ans.Ns)
	FromRRs(&msg.Additional, &ans.Extra)

	opt := quiz.IsEdns0()
	if opt != nil {
		for _, o := range opt.Option {
			if e, ok := o.(*dns.EDNS0_SUBNET); ok {
				msg.Edns_client_subnet = fmt.Sprintf("%s/%d", e.Address.String(), e.SourceNetmask)
				break
			}
		}
	}

	return
}

func FromRRs(jrs *[]DNSRR, rrs *[]dns.RR) {
	*jrs = make([]DNSRR, 0)
	for _, rr := range *rrs {
		jr := FromRR(rr)
		if jr != nil {
			*jrs = append(*jrs, *jr)
		}
	}
}

func (jr *DNSRR) Translate() (rr dns.RR, err error) {
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
	case dns.TypeOPT:
		rr = &dns.OPT{} // FIXME: dummy
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
	default:
		// panic(fmt.Sprintf("unknown type %s", jr.Type))
		err = ErrBadQtype
		return
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

func FromRR(rr dns.RR) (jr *DNSRR) {
	jr = &DNSRR{
		Name: rr.Header().Name,
		Type: int32(rr.Header().Rrtype),
		TTL:  int32(rr.Header().Ttl),
	}
	switch v := rr.(type) {
	case *dns.A:
		jr.Data = v.A.String()
	case *dns.NS:
		jr.Data = v.Ns
	case *dns.MD:
		jr.Data = v.Md
	case *dns.MF:
		jr.Data = v.Mf
	case *dns.CNAME:
		jr.Data = v.Target
	case *dns.SOA:
		jr.Data = fmt.Sprintf("%s %s %d %d %d %d %d", v.Ns, v.Mbox, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl)
	case *dns.MB:
		jr.Data = v.Mb
	case *dns.MG:
		jr.Data = v.Mg
	case *dns.MR:
		jr.Data = v.Mr
	case *dns.NULL:
	case *dns.PTR:
		jr.Data = v.Ptr
	case *dns.HINFO:
	case *dns.MINFO:
	case *dns.MX:
		jr.Data = fmt.Sprintf("%d %s", v.Preference, v.Mx)
	case *dns.TXT:
		jr.Data = strings.Join(v.Txt, " ")
	case *dns.RP:
		jr.Data = fmt.Sprintf("%s %s", v.Mbox, v.Txt)
	case *dns.AAAA:
		jr.Data = v.AAAA.String()
	case *dns.SRV:
		jr.Data = fmt.Sprintf("%d %d %d %s", v.Priority, v.Weight, v.Port, v.Target)
	case *dns.SPF:
		jr.Data = strings.Join(v.Txt, " ")
	case *dns.DS:
		jr.Data = fmt.Sprintf("%d %d %d %s", v.KeyTag, v.Algorithm, v.DigestType, v.Digest)
	case *dns.SSHFP:
		jr.Data = fmt.Sprintf("%d %d %s", v.Algorithm, v.Type, v.FingerPrint)
	case *dns.RRSIG:
		jr.Data = fmt.Sprintf("%d %d %d %d %d %d %s %s", v.Algorithm, v.Labels, v.OrigTtl, v.Expiration, v.Inception, v.KeyTag, v.SignerName, v.Signature)
	case *dns.NSEC:
		var datas []string = make([]string, 1)
		datas[0] = fmt.Sprintf("%s", v.NextDomain)
		for _, b := range v.TypeBitMap {
			if s, ok := dns.TypeToString[b]; ok {
				datas = append(datas, s)
			}
		}
		jr.Data = strings.Join(datas, " ")
	case *dns.DNSKEY:
		jr.Data = fmt.Sprintf("%d %d %d %s", v.Flags, v.Protocol, v.Algorithm, v.PublicKey)
	case *dns.NSEC3:
		var datas []string = make([]string, 1)
		datas[0] = fmt.Sprintf("%d %d %d %d %s %d %s", v.Hash, v.Flags, v.Iterations, v.SaltLength, v.SaltLength, v.HashLength, v.NextDomain)
		for _, b := range v.TypeBitMap {
			if s, ok := dns.TypeToString[b]; ok {
				datas = append(datas, s)
			}
		}
		jr.Data = strings.Join(datas, " ")
	case *dns.NSEC3PARAM:
		jr.Data = fmt.Sprintf("%d %d %d %d %s", v.Hash, v.Flags, v.Iterations, v.SaltLength, v.Salt)
	}
	return
}
