#!/usr/bin/python3
# -*- coding: utf-8 -*-
'''
@date: 2020-05-24
@author: Shell.Xu
@copyright: 2020, Shell.Xu <shell909090@gmail.com>
@license: BSD-3-clause
'''
import sys
import csv
import pprint
import random
import ipaddress
import subprocess

from multiprocessing.pool import ThreadPool

# 101.0.0.0/8 // sh telecom
# 52.88.0.0/13 // aws canada
# 104.244.40.0/21 // twitter


def result_parse(p):
    for rec in p.stdout.decode('utf-8').splitlines():
        try:
            yield ipaddress.ip_address(rec.strip())
        except ValueError:
            pass


def check_poisoned(prot, url):
    cidr = ipaddress.ip_network('104.244.40.0/21')
    p = subprocess.run(['bin/doh', '-short', '-driver', prot, '-s', url, 'www.twitter.com'], capture_output=True)
    return not all([(ip in cidr) for ip in result_parse(p)])


def test_latency(prot, url):
    latency = []
    for _ in range(5):
        p = subprocess.run(['bin/doh', '-driver', prot, '-s', url, 'www.baidu.com'], capture_output=True)
        for rec in p.stdout.decode('utf-8').splitlines():
            if not rec.startswith(";; Query time:"):
                continue
            latency.append(int(rec.strip().split(':', 1)[1].split()[0]))
    if len(latency) != 0:
        return sum(latency)/len(latency)


def test_edns_subnet(prot, url):
    p = subprocess.run(['bin/doh', '-short', '-driver', prot, '-s', url, '-subnet', '101.80.0.0', 'www.taobao.com'], capture_output=True)
    ips1 = set([ipaddress.IPv4Network(ip).supernet(16) for ip in result_parse(p)])
    p = subprocess.run(['bin/doh', '-short', '-driver', prot, '-s', url, '-subnet', '52.88.0.0', 'www.taobao.com'], capture_output=True)
    ips2 = set([ipaddress.IPv4Network(ip).supernet(16) for ip in result_parse(p)])
    return ips1 != ips2


accuracy_domains = ['www.taobao.com', 'www.qq.com', 'www.baidu.com', 'www.meituan.com', 'www.jd.com']
def test_accuracy_inChina(prot, url):
    score = 0
    for domain in accuracy_domains:
        p = subprocess.run(['bin/doh', '-short', '-driver', prot, '-s', url, domain], capture_output=True)
        if most_match[domain] == set(result_parse(p)):
            score += 1
    return score


def test_all(name, prot, url):
    driver = prot
    if prot in ('udp', 'tcp', 'tls'):
        driver = 'dns'
    latency = test_latency(driver, url)
    if latency is None:
        writer.writerow((name, prot, 'not available', '', '', ''))
        return
    poisoned = 'Yes' if check_poisoned(driver, url) else 'No'
    edns_subnet = 'Yes' if test_edns_subnet(driver, url) else 'No'
    accuracy = test_accuracy_inChina(driver, url)
    writer.writerow((name, prot, latency, poisoned, edns_subnet, accuracy))


def main():
    global most_match
    most_match = {}
    for domain in accuracy_domains:
        p = subprocess.run(['bin/doh', '-short', '-s', 'udp://114.114.114.114', domain], capture_output=True)
        most_match[domain] = set(result_parse(p))

    with open(sys.argv[1]) as fi:
        reader = csv.reader(fi)
        servers = [row for row in reader]

    random.shuffle(servers)

    global writer
    writer = csv.writer(sys.stdout)

    pool = ThreadPool(5)
    for row in servers:
        pool.apply_async(test_all, row)
    pool.close()
    pool.join()


if __name__ == '__main__':
    main()
