# Table of content

* [Abstract](#abstract)
* [Compile and Install](#compile-and-install)
* [Command line options and args](#command-line-options-and-args)
* [Config](#config)
  * [Client Config](#client-config)
* [Drivers and Protocols](#drivers-and-protocols)
  * [dns](#dns)
  * [doh](#doh)
* [Public recursive server](#public-recursive-server)
  * [114](#114)
  * [Cloudflare one](#cloudflare-one)
  * [Cloudflare doh](#cloudflare-doh)
  * [Google](#google)
  * [OpenDNS](#opendns)
  * [Quad9](#quad9)
  * [NextDNS](#nextdns)
* [Config](#config)
* [Suggestions in China](#suggestions-in-china)
* [TODO](#todo)

# Abstract

[DNS over HTTPS](https://en.wikipedia.org/wiki/DNS_over_HTTPS) utils written by golang.

# Compile and Install

	make

The executable file is placed under `bin/`. Copy it to wherever you like. Enjoy.

# Command line options and args

See `doh --help`.

# Config

* logfile: optional. indicate which file log should be written to. empty means stdout. empty by default.
* loglevel: optional. log level. warning by default.
* service-driver: optional. see "drivers and protocols". if empty, program will auto guess.
* service-url: required. see "drivers and protocols".
* cert-file: optional. cert file when use doh.
* key-file: optional. key file when use doh.
* edns-client-subnet: optional. it could be empty, means don't do anything. or "client", means read remote address and put it into edns-client-subnet. or an ip address/cidr subnet, means put this address into edns-client-subnet. empty by default.
* client: client settings.
* aliases: a dict. if url matches the key, it will be replaced by value.

## Client Config

* driver: optional. determine which system will be used as a client. see "drivers and protocols". if empty, the program will auto guess.
* url: required. see "drivers and protocols".
* insecure: optional. don't verify the certificate from the server.

# Drivers and Protocols

## dns

There have three different protocols in driver DNS:

* udp
* tcp
* tcp-tls

Here are some examples as output.

	doh -q --url udp://114.114.114.114:53 www.baidu.com
	doh -q --url tcp://114.114.114.114:53 www.baidu.com
	doh -q --url tcp-tls://one.one.one.one:853 www.baidu.com

Here are some examples as input.

	doh --config data/udp-rfc8484.json
	dig www.baidu.com @127.0.0.1 -p 5053

## doh

DoH means DNS over HTTPS. It include two drivers:

* rfc8484
* google

As an output driver, you should indicate which driver exactly. We will guess if you don't say it explicitly.

Here are some examples as output.

	doh -q --url https://security.cloudflare-dns.com/dns-query www.baidu.com
	doh -q --url https://dns.google.com/resolve www.baidu.com

As an input protocol, doh are fine. We support both protocols on the same http/https server.

Here are some examples as input.

	doh --config data/http.json &
	doh -q --url http://localhost:8053/dns-query www.baidu.com
	doh -q --url http://localhost:8053/resolve www.baidu.com

	doh --config data/https.json &
	doh -q --url https://localhost:8153/dns-query --insecure www.baidu.com
	doh -q --url https://localhost:8153/resolve --insecure www.baidu.com

# Public recursive server

Generally speaking, udp are accessible in China, but the results will been poisioned. tcp has been blocked. I won't repeat those in the next lines.

## 114

* Domain: public1.114dns.com
* IP: 114.114.114.114
* Accept protocols: udp/tcp
* Response time in Shanghai: udp:12, tcp:22
* Don't accept edns-client-subnet, in any protocols.

## Cloudflare one

* Domain: one.one.one.one
* IP: 1.1.1.1/1.0.0.1
* Accept protocols: udp/tcp/tcp-tls
* Response time in Shanghai: udp:78, tcp:157, tcp-tls:250
* Response time in Japan IDC: udp:2, tcp:2, tcp-tls:22
* Don't accept edns-client-subnet, in any protocols.
* In China: tcp-tls accessible.
* Accuracy: not best result in China.

## Cloudflare doh

* Domain: security.cloudflare-dns.com
* IP: 104.18.2.55/104.18.3.55
* Accept protocols: rfc8484
* Response time in Shanghai: rfc8484:83
* Response time in Japan IDC: rfc8484:4
* Don't accept edns-client-subnet, in any protocols.
* In China: rfc8484 accessible.
* Accuracy: not best result in China.

## Google

* Domain: dns.google.com
* IP: 8.8.8.8/8.8.4.4
* Accept protocols: udp/tcp/tcp-tls/google.
* Response time in Shanghai: udp:75, tcp:90, tcp-tls:250
* Response time in Japan IDC: udp:1, tcp:2, tcp-tls:40, google:5-100
* Accept edns-client-subnet with protocol google. Sometimes edns-client-subnet work with udp, dependence on which server you actually use.
* In China: tcp-tls accessible. `google` need a proxy.
* A proxy will be needed for protocol google in China.

## OpenDNS

* Domain: dns.opendns.com, doh.opendns.com (for rfc8484)
* IP: 208.67.222.222/208.67.220.220
* Accept protocols: udp/tcp/rfc8484
* Response time in Shanghai: udp:80, tcp:156, rfc8484:400
* Response time in Japan IDC: udp:1, tcp:2, rfc8484:80
* Don't accept edns-client-subnet, in any protocols.
* In China: rfc8484 accessible.

## Quad9

* Domain: dns.quad9.net
* IP: 9.9.9.9/149.112.112.112
* Accept protocols: udp/tcp/tcp-tls/rfc8484
* Response time in Shanghai: udp:85, tcp:170, tcp-tls:280, rfc8484: 80
* Response time in Japan IDC: udp:115, tcp:250, tcp-tls:370, rfc8484: 1
* Don't accept edns-client-subnet, in any protocols.
* In China: tcp-tls accessible. rfc8484 accessible.
* Accuracy: wrong result in China (taobao and baidu).

## AdGuard

* Domain: dns.adguard.com
* IP: 176.103.130.130/176.103.130.131
* Accept protocols: udp/tcp/tcp-tls/rfc8484
* Response time in Shanghai: udp:80, tcp:140, tcp-tls:280, rfc8484: 370
* Response time in Japan IDC: udp:1, tcp:2, tcp-tls:70, rfc8484: 70
* Don't accept edns-client-subnet, in any protocols.
* In China: tcp-tls accessible. rfc8484 accessible.
* Accuracy: wrong result in China (taobao and baidu).

## NextDNS

* Domain: dns.nextdns.io, 76dc6f.dns.nextdns.io (for tcp-tls)
* IP: 45.90.28.253/45.90.30.253
* Accept protocols: udp/tcp/tcp-tls/rfc8484
* Response time in Shanghai: udp:50, tcp:100, tcp-tls:190, rfc8484: 192
* Response time in Japan IDC: udp:1, tcp:2, tcp-tls:85, rfc8484: 100
* Don't accept edns-client-subnet, in any protocols.
* In China: tcp-tls accessible. rfc8484 accessible.
* Accuracy: wrong result in China (taobao and baidu).

# Suggestions in China

1. Don't use Quad9, NextDNS, and AdGuard. Wrong results means useless.
2. I won't suggest Cloudflare. Not the best result. Don't use it unless running out of other options.
3. My first option is Google with tcp-tls. Accessible, accurate, easy.
4. Secend option is Google/OpenDNS with udp. Find yourself a way to dodge the firewall.
4. Google with edns-client-subnet (proxy needed) are barely acceptable.
5. If you want tcp-tls, the first choice is Google, then Cloudflare.
6. If you want rfc8484, the first choice is OpenDNS, then Cloudflare.

# TODO

* cache
* record dns logs
* multiple outputs, load balance

https://blog.skk.moe/post/which-public-dns-to-use/
https://github.com/curl/curl/wiki/DNS-over-HTTPS
https://www.publicdns.xyz/
