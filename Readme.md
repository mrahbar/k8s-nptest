Based on: [kubernetes/perf-tests](https://github.com/kubernetes/perf-tests/blob/master/network/benchmarks/netperf)

Docker Image: [here](https://hub.docker.com/r/endianogino/netperf/)

Helm chart: [here](https://github.com/mrahbar/k8s-testsuite)
# Benchmarking Kubernetes Networking Performance

## Objectives:
A standardized benchmark to measure Kubernetes networking performance on multiple host platforms and network stacks.

## Implementation and Test Methodology

The benchmark can be executed via a single Go binary invocation that triggers all the automated testing located in the orchestrator and worker pods as seen below. The test uses a custom docker container that has the go binary and iperf3 and other tools built into it. 
The orchestrator pod coordinates the worker pods to run tests in serial order for the 4 scenarios described below, at MTUs (MSS tuning for TCP and direct packet size tuning for UDP). The MTU range covers 96 till 1460 in steps of 64.

Using node labels, the Worker Pods 1 and 2 are placed on the same Kubernetes node, and Worker Pod 3 is placed on a different node. The nodes all communicate with the orchestrator pod service using simple golang rpcs and request work items. A minimum of two Kubernetes worker nodes are necessary for this test.

The 5 major network traffic paths are combination of Pod IP vs Virtual IP and whether the pods are co-located on the same node/VM versus a remotely located pod.

* Same VM using Pod IP: Same VM Pod to Pod traffic tests from Worker 1 to Worker 2 using its Pod IP.

* Same VM using Cluster/Virtual IP: Same VM Pod to Pod traffic tests from Worker 1 to Worker 2 using its Service IP (also known as its Cluster IP or Virtual IP).

* Remote VM using Pod IP: Worker 3 to Worker 2 traffic tests using Worker 2 Pod IP.

* Remote VM using Cluster/Virtual IP: Worker 3 to Worker 2 traffic tests using Worker 2 Cluster/Virtual IP.

* Same VM Pod Hairpin: Worker 2 to itself using Cluster IP

The orchestrator and worker pods run independently of the initiator script, with the orchestrator pod sending work items to workers till the testcase schedule is complete.
The iperf output (both TPC and UDP modes) and the netperf TCP output from all worker nodes is uploaded to the orchestrator pod where it is filtered and the results are written to the output file as well as to stdout log.
Default file locations are /tmp/result.csv and /tmp/output.txt for the raw results.

## Output Raw CSV data
**All units in the csv file are in Gbits/second**
```console
ALL TESTCASES AND MSS RANGES COMPLETE - GENERATING CSV OUTPUT
the output for each MSS testpoint is a single value in Gbits/sec 
MSS , Maximum, 96, 160, 224, 288, 352, 416, 480, 544, 608, 672, 736, 800, 864, 928, 992, 1056, 1120, 1184, 1248, 1312, 1376, 1460
1 iperf TCP. Same VM using Pod IP ,24252.000000,22650,23224,24101,23724,23532,23092,23431,24102,24072,23431,23871,23897,23275,23146,23535,24252,23662,22133,,23514,23796,24008,
2 iperf TCP. Same VM using Virtual IP ,26052.000000,26052,0,25382,23702,0,22703,22549,0,23085,22074,0,22366,23516,0,23059,22991,0,23231,22603,0,23255,23605,
3 iperf TCP. Remote VM using Pod IP ,910.000000,239,426,550,663,708,742,769,792,811,825,838,849,859,866,874,883,888,894,898,903,907,910,
4 iperf TCP. Remote VM using Virtual IP ,906.000000,0,434,546,0,708,744,0,791,811,0,837,849,0,868,875,0,888,892,0,903,906,0,
5 iperf TCP. Hairpin Pod to own Virtual IP ,23493.000000,22798,21629,0,22159,21132,0,22900,21816,0,21775,21425,0,22172,21611,21869,22865,22003,22562,23493,22684,217872,
6 iperf UDP. Same VM using Pod IP ,6647.000000,6647,
7 iperf UDP. Same VM using Virtual IP ,6554.000000,6554,
8 iperf UDP. Remote VM using Pod IP ,1877.000000,1877,
9 iperf UDP. Remote VM using Virtual IP ,1695.000000,1695,
10 netperf. Same VM using Pod IP ,7003.430000,7003.43,
11 netperf. Same VM using Virtual IP ,0.000000,0.00,
12 netperf. Remote VM using Pod IP ,908.460000,908.46,
13 netperf. Remote VM using Virtual IP ,0.000000,0.00,
END CSV DATA
```