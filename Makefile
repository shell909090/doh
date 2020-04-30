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

test: test-query test-udp-rfc8484 test-http test-https

test-query: build
	bin/doh -q --short --protocol dns --url udp://114.114.114.114:53 www.baidu.com
	bin/doh -q --short --protocol dns --url tcp://114.114.114.114:53 www.baidu.com
	bin/doh -q --short --protocol dns --url tcp-tls://one.one.one.one:853 www.baidu.com
	bin/doh -q --short --protocol rfc8484 --url https://security.cloudflare-dns.com/dns-query www.baidu.com
	bin/doh -q --short --protocol google --url https://dns.google.com/resolve www.baidu.com

	bin/doh -q --short --protocol dns --url udp://114.114.114.114:53 www.google.com
	bin/doh -q --short --protocol dns --url tcp://114.114.114.114:53 www.google.com
	bin/doh -q --short --protocol dns --url tcp-tls://one.one.one.one:853 www.google.com
	bin/doh -q --short --protocol rfc8484 --url https://security.cloudflare-dns.com/dns-query www.google.com
	bin/doh -q --short --protocol google --url https://dns.google.com/resolve www.google.com

test-udp-rfc8484: bin/doh
	bin/doh --config udp-rfc8484.json &
	sleep 1
	dig +short www.google.com @127.0.0.1 -p 5053
	dig +short +subnet=101.80.0.0 www.google.com @127.0.0.1 -p 5053
	dig +short +subnet=104.244.42.1 www.google.com @127.0.0.1 -p 5053
	killall doh

test-udp-google: bin/doh
	bin/doh --config udp-google.json &
	sleep 1
	dig +short www.google.com @127.0.0.1 -p 5153
	dig +short +subnet=101.80.0.0 www.google.com @127.0.0.1 -p 5153
	dig +short +subnet=104.244.42.1 www.google.com @127.0.0.1 -p 5153
	killall doh

test-http: bin/doh
	bin/doh --config doh.json &
	sleep 1
	bin/doh -q --short --protocol rfc8484 --url http://localhost:8053/dns-query www.baidu.com
	killall doh

test-https: bin/doh
	bin/doh --config dohs.json &
	sleep 1
	bin/doh -q --short --protocol rfc8484 --url https://localhost:8153/dns-query --insecure www.baidu.com
	killall doh

### Makefile ends here
