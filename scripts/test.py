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
import random
import ipaddress
import subprocess

import pprint

from multiprocessing.pool import ThreadPool

# 101.0.0.0/8 // sh telecom
# 52.88.0.0/13 // aws canada
# 104.244.40.0/21 // twitter

accuracy_domains = [
    'www.taobao.com', 'www.tmall.com', 'www.qq.com', 'www.baidu.com', 'www.sohu.com', 'www.jd.com',
    'www.amazon.com', 'www.bing.com', 'www.linkedin.com', 'www.weibo.com', 'www.meituan.com']


def result_parse(p):
    for rec in p.stdout.decode('utf-8').splitlines():
        try:
            rec = rec.strip()
            yield ipaddress.ip_address(rec)
        except ValueError:
            if 'ERROR' in rec:
                print(rec, file=sys.stderr)


# def cidr_result(ips):
#     return set([ipaddress.IPv4Network(ip).supernet(8) for ip in ips])


ping_cache = {}
def ping(ip):
    if ip in ping_cache:
        return ping_cache[ip]
    p = subprocess.run(
        ['ping', '-qc', '10', '-i', '0.2', str(ip)],
        capture_output=True)
    for line in p.stdout.decode('utf-8').splitlines():
        line = line.strip()
        if not line.startswith('rtt'):
            continue
        rtt = line.split('=')[1].split('/')
        ping_cache[ip] = float(rtt[1])
        break
    if ip not in ping_cache:
        ping_cache[ip] = 10000
    return ping_cache[ip]


# def get_best_answer(urls, domain):
#     answers = set()
#     for url in urls:
#         p = subprocess.run(
#             ['bin/doh', '-short', '-s', url, domain], capture_output=True)
#         for ip in result_parse(p):
#             answers.add(ip)
#     for ip in answers:
#         print(ip, ping(ip))


def repeat_ips(num, cmd):
    ips = set()
    for _ in range(num):
        p = subprocess.run(cmd, capture_output=True)
        ips |= set(result_parse(p))
    return ips


def test_available(row):
    name, prot, url = row
    driver = prot
    if prot in ('udp', 'tcp', 'tls'):
        driver = 'dns'
    ips = repeat_ips(2, ['bin/doh', '-short', '-insecure', '-driver', driver, '-s', url, 'www.amazon.com'])
    if not ips:
        writer.writerow((name, prot, 'not available'))
    else:
        return row


accuracies = {}
def test_accuracy(domain):
    global accuracies
    rslt = {}
    avg_latencies = []
    min_latencies = []
    for row in servers:
        name, prot, url = row
        driver = prot
        if prot in ('udp', 'tcp', 'tls'):
            driver = 'dns'
        latency = [ping(ip) for ip in repeat_ips(3, [
            'bin/doh', '-short', '-insecure', '-driver', driver, '-s', url, domain])]
        if not latency:
            print(name, prot, domain, 'fuck off', file=sys.stderr)
            avg_latencies.append(10000)
            continue
        min_latencies.append(min(latency))
        avg_latencies.append(sum(latency) / len(latency))
    min_latency = min(min_latencies)
    for row, latency in zip(servers, avg_latencies):
        rslt[row] = latency/min_latency
    accuracies[domain] = rslt


def test_latency(driver, url):
    latency = []
    for _ in range(5):
        p = subprocess.run(
            ['bin/doh', '-driver', driver, '-s', url, 'www.amazon.com'],
            capture_output=True)
        for rec in p.stdout.decode('utf-8').splitlines():
            if not rec.startswith(";; Query time:"):
                continue
            latency.append(int(rec.strip().split(':', 1)[1].split()[0]))
    if len(latency) != 0:
        return sum(latency)/len(latency)


def check_poisoned(driver, url):
    cidr = ipaddress.ip_network('104.244.40.0/21')
    p = subprocess.run(
        ['bin/doh', '-short', '-insecure', '-driver', driver, '-s', url, 'www.twitter.com'],
        capture_output=True)
    return not all([(ip in cidr) for ip in result_parse(p)])


def test_edns_subnet(driver, url):
    ips1 = repeat_ips(5, ['bin/doh', '-short', '-insecure', '-driver', driver, '-s', url,
                          '-subnet', '101.80.0.0', 'www.amazon.com'])
    ips2 = repeat_ips(5, ['bin/doh', '-short', '-insecure', '-driver', driver, '-s', url,
                          '-subnet', '52.88.0.0', 'www.amazon.com'])
    return bool(ips1) and bool(ips2) and not bool(ips1 & ips2)


# def test_accuracy_inChina(driver, url):
#     r = {}
#     for domain in accuracy_domains:
#         r[domain] = repeat_ips(5, ['bin/doh', '-short', '-driver', driver, '-s', url, domain])
#     return r


def test_all(row):
    name, prot, url = row
    driver = prot
    if prot in ('udp', 'tcp', 'tls'):
        driver = 'dns'
    latency = test_latency(driver, url)
    if latency is None:
        writer.writerow((name, prot, 'not available'))
        return
    poisoned = 'Yes' if check_poisoned(driver, url) else 'No'
    edns_subnet = 'Yes' if test_edns_subnet(driver, url) else 'No'
    accuracy = [accuracies[domain][row] for domain in accuracy_domains]
    rslt = [name, prot, latency, poisoned, edns_subnet, '%0.3f' % sum(accuracy)]
    writer.writerow(rslt + ['%0.2f' % score for score in accuracy])


# def translate_accuracy(pool, rslts):
#     all_ips, min_ips = {}, {}
#     for _, _, accuracy in rslts:
#         for domain, ips in accuracy.items():
#             all_ips[domain] = set(ips) | all_ips.get(domain, set())
#     for domain, ips in all_ips.items():
#         min_ips[domain] = min(pool.map(ping, ips))
#     for row in rslts:
#         score = 0
#         for domain, ips in row[2].items():
#             latency = [ping(ip) for ip in ips]
#             avg = sum(latency) / len(latency)
#             score += avg/min_ips[domain]
#         row = list(row)
#         row[2] = score
#         yield row


def main():
    global writer
    writer = csv.writer(sys.stdout)

    global servers
    with open(sys.argv[1]) as fi:
        servers = [tuple(row) for row in csv.reader(fi)]
    random.shuffle(servers)

    pool = ThreadPool(5)

    servers = [row for row in pool.map(test_available, servers) if row]
    pool.map(test_accuracy, accuracy_domains)
    pool.map(test_all, servers)


if __name__ == '__main__':
    main()
