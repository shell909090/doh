package iplist

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"testing"

	logging "github.com/op/go-logging"
)

const (
	IPLIST = "10.0.0.0 255.0.0.0\n172.16.0.0 255.240.0.0\n192.168.0.0 255.255.0.0"
)

func SetLogging(logfile, loglevel string) (err error) {
	var file *os.File
	file = os.Stdout

	if loglevel == "" {
		loglevel = "WARNING"
	}
	if logfile != "" {
		file, err = os.OpenFile(logfile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			panic(err.Error())
		}
	}
	logging.SetBackend(logging.NewLogBackend(file, "", 0))
	logging.SetFormatter(logging.MustStringFormatter(
		"%{time:01-02 15:04:05.000}[%{level}] %{shortpkg}/%{shortfile}: %{message}"))
	lv, err := logging.LogLevel(loglevel)
	if err != nil {
		panic(err.Error())
	}
	logging.SetLevel(lv, "")
	return
}

func init() {
	SetLogging("", "ERROR")
}

func genIP() (ip net.IP) {
	buf := make([]byte, 4)
	_, err := io.ReadFull(crand.Reader, buf)
	if err != nil {
		panic("no rand.")
	}
	return net.IP(buf)
}

func genMask() (mask net.IPMask, n int) {
	n = 1 + rand.Intn(31)
	mask = net.CIDRMask(n, 32)
	return
}

func TestReadIPList(t *testing.T) {
	var buf bytes.Buffer

	for i := 0; i < 100; i++ {
		mask, _ := genMask()
		fmt.Fprintf(&buf, "%s %s\n", genIP().String(), net.IP(mask).String())
	}

	_, err := ReadIPList(&buf)
	if err != nil {
		t.Error(err)
		return
	}
	return
}

func BenchmarkReadIPList(b *testing.B) {
	var buf bytes.Buffer

	for i := 0; i < b.N; i++ {
		mask, _ := genMask()
		fmt.Fprintf(&buf, "%s %s\n", genIP().String(), net.IP(mask).String())
	}
	b.ResetTimer()

	_, err := ReadIPList(&buf)
	if err != nil {
		b.Error(err)
		return
	}
	return
}

func TestReadIPListCIDR(t *testing.T) {
	var buf bytes.Buffer

	for i := 0; i < 100; i++ {
		_, n := genMask()
		fmt.Fprintf(&buf, "%s/%d\n", genIP().String(), n)
	}

	_, err := ReadIPList(&buf)
	if err != nil {
		t.Error(err)
		return
	}
	return
}

func BenchmarkReadIPListCIDR(b *testing.B) {
	var buf bytes.Buffer

	for i := 0; i < b.N; i++ {
		_, n := genMask()
		fmt.Fprintf(&buf, "%s/%d\n", genIP().String(), n)
	}
	b.ResetTimer()

	_, err := ReadIPList(&buf)
	if err != nil {
		b.Error(err)
		return
	}
	return
}

func TestIPList(t *testing.T) {
	buf := bytes.NewBufferString(IPLIST)
	filter, err := ReadIPList(buf)
	if err != nil {
		t.Fatalf("ReadIPList failed: %s", err)
	}

	if !filter.Contain(net.ParseIP("192.168.1.1")) {
		t.Fatalf("Contain wrong1.")
	}

	if !filter.Contain(net.ParseIP("10.8.0.1")) {
		t.Fatalf("Contain wrong2.")
	}

	if filter.Contain(net.ParseIP("211.80.90.25")) {
		t.Fatalf("Contain wrong3.")
	}
}

func BenchmarkIPList(b *testing.B) {
	var buf bytes.Buffer

	for i := 0; i < b.N/1000; i++ {
		_, n := genMask()
		fmt.Fprintf(&buf, "%s/%d\n", genIP().String(), n)
	}

	filter, err := ReadIPList(&buf)
	if err != nil {
		b.Error(err)
		return
	}
	ip := genIP()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.Contain(ip)
	}
	return
}

func BenchmarkIPListBig(b *testing.B) {
	var buf bytes.Buffer

	for i := 0; i < b.N; i++ {
		_, n := genMask()
		fmt.Fprintf(&buf, "%s/%d\n", genIP().String(), n)
	}

	filter, err := ReadIPList(&buf)
	if err != nil {
		b.Error(err)
		return
	}
	ip := genIP()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.Contain(ip)
	}
	return
}

func BenchmarkIPListRand(b *testing.B) {
	var buf bytes.Buffer

	for i := 0; i < b.N/1000; i++ {
		_, n := genMask()
		fmt.Fprintf(&buf, "%s/%d\n", genIP().String(), n)
	}

	filter, err := ReadIPList(&buf)
	if err != nil {
		b.Error(err)
		return
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		filter.Contain(genIP())
	}
	return
}
