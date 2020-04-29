# Abstract

[DNS over HTTPS](https://en.wikipedia.org/wiki/DNS_over_HTTPS) utils written by golang.

# Examples

	doh --query --protocol dns --url udp://114.114.114.114:53 www.baidu.com
	doh --query --protocol dns --url tcp://114.114.114.114:53 www.baidu.com
	doh --query --protocol dns --url tcp-tls://one.one.one.one:853 www.baidu.com

	doh --query --protocol rfc8484 --url https://security.cloudflare-dns.com/dns-query www.baidu.com
	# with proxy
	doh --query --protocol google --url https://dns.google.com/resolve www.baidu.com

	doh --listen udp://127.0.0.1:5053 --protocol rfc8484 --url https://security.cloudflare-dns.com/dns-query
	dig www.baidu.com @127.0.0.1 -p 5053

	doh --serve doh --listen http://127.0.0.1:8080 --protocol dns --url udp://114.114.114.114:53
	doh --query --protocol rfc8484 --url http://localhost:8080/dns-query www.baidu.com
