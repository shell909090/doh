package drivers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/miekg/dns"
)

func WriteFull(w io.Writer, b []byte) (err error) {
	n, err := w.Write(b)
	if err != nil {
		return
	}
	if n != len(b) {
		return io.ErrShortWrite
	}
	return
}

type Rfc8484Client struct {
	URL       string
	Insecure  bool
	Timeout   int
	transport *http.Transport
}

func NewRfc8484Client(URL string, body json.RawMessage) (cli *Rfc8484Client) {
	cli = &Rfc8484Client{}
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

func (cli *Rfc8484Client) Url() (u string) {
	return cli.URL
}
func (cli *Rfc8484Client) Exchange(ctx context.Context, quiz *dns.Msg) (ans *dns.Msg, err error) {
	bquiz, err := quiz.Pack()
	if err != nil {
		logger.Error(err.Error())
		return
	}

	if cli.Timeout != 0 {
		ctx, _ = context.WithTimeout(ctx, time.Duration(cli.Timeout)*time.Millisecond)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", cli.URL, bytes.NewBuffer(bquiz))
	if err != nil {
		logger.Error(err.Error())
		return
	}
	req.Header.Add("Accept", "application/dns-message")
	req.Header.Add("Content-Type", "application/dns-message")

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

	bbody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	ans = &dns.Msg{}
	err = ans.Unpack(bbody)
	if err != nil {
		logger.Error(err.Error())
		return
	}

	return
}

type Rfc8484Handler struct {
	EdnsClientSubnet string
	cli              Client
}

func NewRfc8484Handler(cli Client, EdnsClientSubnet string) (handler *Rfc8484Handler) {
	handler = &Rfc8484Handler{
		EdnsClientSubnet: EdnsClientSubnet,
		cli:              cli,
	}
	return
}

func (handler *Rfc8484Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	defer req.Body.Close()

	var bdns []byte
	switch req.Method {
	case "GET":
		err = req.ParseForm()
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		b64dns := req.Form.Get("dns")
		bdns, err = base64.StdEncoding.DecodeString(b64dns)
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

	case "POST":
		bdns, err = ioutil.ReadAll(req.Body)
		if err != nil {
			logger.Error(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}

	default:
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	quiz := &dns.Msg{}
	err = quiz.Unpack(bdns)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = HttpSetEdns0Subnet(w, req, "", handler.EdnsClientSubnet, quiz)
	if err != nil {
		return
	}

	logger.Infof("rfc8484 server query: %s", quiz.Question[0].Name)

	ctx := context.Background()
	ans, err := handler.cli.Exchange(ctx, quiz)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	bdns, err = ans.Pack()
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	w.Header().Add("Content-Type", "application/dns-message")
	w.Header().Add("Cache-Control", "no-cache, max-age=0")
	w.WriteHeader(http.StatusOK)

	err = WriteFull(w, bdns)
	if err != nil {
		logger.Error(err.Error())
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	return
}
