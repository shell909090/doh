### Makefile --- 

## Author: shell@dsk
## Version: $Id: Makefile,v 0.0 2020/04/29 14:02:48 shell Exp $
## Keywords: 
## X-URL: 
LEVEL=NOTICE

all: clean build

clean:
	rm -rf bin pkg gopath debuild *.log

build: bin/doh

bin/doh:
	mkdir -p gopath/src/github.com/shell909090/
	ln -s "$$PWD" gopath/src/github.com/shell909090/doh
	mkdir -p bin
	GOPATH="$$PWD/gopath":"$$GOPATH" go build -o bin/doh github.com/shell909090/doh/doh
	rm -rf gopath

install: bin/doh
	install -d $(DESTDIR)/usr/bin/
	install -m 755 -s bin/doh $(DESTDIR)/usr/bin/

test:
	go test -v github.com/shell909090/doh/iplist

benchmark:
	go test -v github.com/shell909090/doh/iplist -bench . -benchmem

build-deb:
	dpkg-buildpackage --no-sign
	mkdir -p debuild
	mv -f ../doh_* debuild

test-query: bin/doh
	bin/doh -q --url udp://114.114.114.114 www.baidu.com
	bin/doh -q --url tcp://114.114.114.114 www.baidu.com
	bin/doh -q --url tcp-tls://one.one.one.one www.baidu.com
	bin/doh -q --url https://security.cloudflare-dns.com/dns-query www.baidu.com
	bin/doh -q --url https://dns.google.com/resolve www.baidu.com
	bin/doh -q --config data/twin.json www.baidu.com

test-short: bin/doh
	bin/doh -q --short --url 114 www.baidu.com
	bin/doh -q --short --url 114t www.baidu.com
	bin/doh -q --short --url one www.baidu.com
	bin/doh -q --short --url cf www.baidu.com
	bin/doh -q --short --url google www.baidu.com
	bin/doh -q --short --config data/twin.json www.baidu.com

test-edns: bin/doh
	bin/doh -q --short --url udp://114.114.114.114 www.google.com
	bin/doh -q --short --subnet 101.80.0.0 --url udp://114.114.114.114 www.google.com
	bin/doh -q --short --subnet 104.244.42.1 --url udp://114.114.114.114 www.google.com
	bin/doh -q --short --url tcp-tls://one.one.one.one www.google.com
	bin/doh -q --short --subnet 101.80.0.0 --url tcp-tls://one.one.one.one www.google.com
	bin/doh -q --short --subnet 104.244.42.1 --url tcp-tls://one.one.one.one www.google.com
	bin/doh -q --short --url https://dns.google.com/resolve www.google.com
	bin/doh -q --short --subnet 101.80.0.0 --url https://dns.google.com/resolve www.google.com
	bin/doh -q --short --subnet 104.244.42.1 --url https://dns.google.com/resolve www.google.com
	bin/doh -q --short --config data/twin.json www.google.com
	bin/doh -q --short --subnet 101.80.0.0 --config data/twin.json www.google.com
	bin/doh -q --short --subnet 104.244.42.1 --config data/twin.json www.google.com

test-rfc8484: bin/doh
	bin/doh --config data/rfc8484.json &
	sleep 1
	dig +short www.google.com @127.0.0.1 -p 5053
	dig +short +subnet=101.80.0.0 www.google.com @127.0.0.1 -p 5053
	dig +short +subnet=104.244.42.1 www.google.com @127.0.0.1 -p 5053
	killall doh

test-google: bin/doh
	bin/doh --config data/google.json &
	sleep 1
	dig +short www.google.com @127.0.0.1 -p 5153
	dig +short +subnet=101.80.0.0 www.google.com @127.0.0.1 -p 5153
	dig +short +subnet=104.244.42.1 www.google.com @127.0.0.1 -p 5153
	killall doh

test-http: bin/doh
	bin/doh --config data/http.json &
	sleep 1
	bin/doh -q --short --url http://localhost:8053/dns-query www.baidu.com
	bin/doh -q --short --url http://localhost:8053/resolve www.baidu.com
	curl -s "http://localhost:8053/resolve?name=www.baidu.com" | jq
	killall doh

test-https: bin/doh
	bin/doh --config data/https.json &
	sleep 1
	bin/doh -q --short --url https://localhost:8153/dns-query --insecure www.baidu.com
	bin/doh -q --short --url https://localhost:8153/resolve --insecure www.baidu.com
	curl -s -k "https://localhost:8153/resolve?name=www.baidu.com" | jq
	killall doh

test-public: bin/doh
	./diagdns www.twitter.com www.baidu.com www.google.com www.taobao.com www.jd.com open.163.com github.com en.wikipedia.org www.startpage.com

### Makefile ends here
