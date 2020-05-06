package iplist

import (
	"bytes"
	crand "crypto/rand"
	"fmt"
	"io"
	"math/rand"
	"net"
	"testing"

	"github.com/shell909090/doh/drivers"
)

const (
	IPLIST = "10.0.0.0 255.0.0.0\n172.16.0.0 255.240.0.0\n192.168.0.0 255.255.0.0"
)

func init() {
	drivers.SetLogging("", "ERROR")
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
