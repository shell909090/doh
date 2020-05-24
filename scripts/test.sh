#!/bin/bash

function query() {
    bin/doh -s udp://114.114.114.114 www.baidu.com
}

function short() {
    bin/doh -short -s 114 www.baidu.com
    bin/doh -short -s 114t www.baidu.com
    bin/doh -short -s one www.baidu.com
    bin/doh -short -s cf www.baidu.com
    bin/doh -short -s google www.baidu.com
    bin/doh -short -config data/twin.json www.baidu.com
}

function edns() {
    bin/doh -short -s udp://114.114.114.114 www.google.com
    bin/doh -short -s udp://114.114.114.114 -subnet 101.80.0.0 www.google.com
    bin/doh -short -s udp://114.114.114.114 -subnet 104.244.42.1 www.google.com
    bin/doh -short -s tcp-tls://one.one.one.one www.google.com
    bin/doh -short -s tcp-tls://one.one.one.one -subnet 101.80.0.0 www.google.com
    bin/doh -short -s tcp-tls://one.one.one.one -subnet 104.244.42.1 www.google.com
    bin/doh -short -s https://dns.google.com/resolve www.google.com
    bin/doh -short -s https://dns.google.com/resolve -subnet 101.80.0.0 www.google.com
    bin/doh -short -s https://dns.google.com/resolve -subnet 104.244.42.1 www.google.com
    bin/doh -short -config data/twin.json -q www.google.com
    bin/doh -short -config data/twin.json -q -subnet 101.80.0.0 www.google.com
    bin/doh -short -config data/twin.json -q -subnet 104.244.42.1 www.google.com
}

function rfc8484() {
    bin/doh -config data/rfc8484.json &
    sleep 1
    bin/doh -short -s udp://127.0.0.1:5053 www.google.com
    bin/doh -short -s udp://127.0.0.1:5053 -subnet=101.80.0.0 www.google.com
    bin/doh -short -s udp://127.0.0.1:5053 -subnet=104.244.42.1 www.google.com
    killall doh
}

function google() {
    bin/doh -config data/google.json &
    sleep 1
    bin/doh -short -s udp://127.0.0.1:5153 www.google.com
    bin/doh -short -s udp://127.0.0.1:5153 -subnet=101.80.0.0 www.google.com
    bin/doh -short -s udp://127.0.0.1:5153 -subnet=104.244.42.1 www.google.com
    killall doh
}

function http() {
    bin/doh -config data/http.json &
    sleep 1
    bin/doh -short -s http://localhost:8053/dns-query www.baidu.com
    bin/doh -short -s http://localhost:8053/resolve www.baidu.com
    curl -s "http://localhost:8053/resolve?name=www.baidu.com" | jq -r '.Answer[].data'
    killall doh
}

function https() {
    bin/doh -config data/https.json &
    sleep 1
    bin/doh -short -s https://localhost:8153/dns-query -insecure www.baidu.com
    bin/doh -short -s https://localhost:8153/resolve -insecure www.baidu.com
    curl -s -k "https://localhost:8153/resolve?name=www.baidu.com" | jq -r '.Answer[].data'
    killall doh
}

$@
