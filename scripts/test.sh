#!/bin/bash

function query() {
    bin/doh -s udp://114.114.114.114 www.baidu.com
}

function short() {
    bin/doh -short -s 114 www.baidu.com
    bin/doh -short -s 114t www.baidu.com
    bin/doh -short -s cftls www.baidu.com
    bin/doh -short -s cfdoh www.baidu.com
    bin/doh -short -s gdnstls www.baidu.com
    bin/doh -short @114 www.baidu.com
    bin/doh -short @114 NS baidu.com
    bin/doh -short @114 IN baidu.com
    bin/doh -short -q -config data/twin.json www.baidu.com
}

function edns() {
    bin/doh -short -s alidns www.taobao.com
    bin/doh -short -s alidns -subnet 101.80.0.0 www.taobao.com
    bin/doh -short -s alidns -subnet 104.244.42.1 www.taobao.com
    bin/doh -short -s dnspod www.taobao.com
    bin/doh -short -s dnspod -subnet 101.80.0.0 www.taobao.com
    bin/doh -short -s dnspod -subnet 104.244.42.1 www.taobao.com
}

function rfc8484() {
    bin/doh -config data/rfc8484.json &
    sleep 1
    bin/doh -short -s udp://127.0.0.1:5053 www.taobao.com
    killall doh
}

function google() {
    bin/doh -config data/google.json &
    sleep 1
    bin/doh -short -s udp://127.0.0.1:5153 www.taobao.com
    killall doh
}

function dnspod() {
    bin/doh -config data/dnspod.json &
    sleep 1
    bin/doh -short -s udp://127.0.0.1:5253 www.taobao.com
    killall doh
}

function http() {
    bin/doh -config data/http.json &
    sleep 1
    bin/doh -short -s http://localhost:8053/dns-query www.baidu.com
    bin/doh -short -s http://localhost:8053/resolve www.baidu.com
    bin/doh -short -s http://localhost:8053/d www.baidu.com
    curl -s "http://localhost:8053/resolve?name=www.baidu.com" | jq -r '.Answer[].data'
    curl -s "http://localhost:8053/d?dn=www.baidu.com&ttl=1"
    echo ""
    killall doh
}

function https() {
    bin/doh -config data/https.json &
    sleep 1
    bin/doh -short -s https://localhost:8153/dns-query -insecure www.baidu.com
    bin/doh -short -s https://localhost:8153/resolve -insecure www.baidu.com
    curl -s -k "https://localhost:8153/resolve?name=www.baidu.com" | jq -r '.Answer[].data'
    curl -s -k "https://localhost:8153/d?dn=www.baidu.com&ttl=1"
    echo ""
    killall doh
}

$@
