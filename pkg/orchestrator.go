package pkg

import (
	"fmt"
	"github.com/mrahbar/k8s-nptest/integration"
	"github.com/mrahbar/k8s-nptest/types"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"regexp"
	"strconv"
	"sync"
)

// Regexes to parse the Mbits/sec out of iperf TCP, UDP and netperf output
var iperfTCPOutputRegexp = regexp.MustCompile("SUM.*\\s+(\\d+)\\sMbits/sec\\s+receiver")
var iperfUDPOutputRegexp = regexp.MustCompile("\\s+(\\S+)\\sMbits/sec\\s+\\S+\\s+ms\\s+")
var netperfOutputRegexp = regexp.MustCompile("\\s+\\d+\\s+\\d+\\s+\\d+\\s+\\S+\\s+(\\S+)\\s*")

var dataPoints = make(map[string][]types.Point)
var dataPointKeys []string
var datapointsFlushed bool

var testcases []*types.Testcase
var currentJobIndex = 0

var globalLock sync.Mutex
var workerStateMap = make(map[string]*types.WorkerState)

const csvSeparator = ";"
const defaultBandwithFailed = "-1"
type NetPerfRpc int

// Blocking RPC server start - only runs on the orchestrator
func Orchestrate(d bool) {
	debug = d
	testcases = []*types.Testcase{
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "1 iperf TCP. Same VM using Pod IP", Type: iperfTcpTest, ClusterIP: false, MSS: mssMin},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "2 iperf TCP. Same VM using Virtual IP", Type: iperfTcpTest, ClusterIP: true, MSS: mssMin},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w3", Label: "3 iperf TCP. Remote VM using Pod IP", Type: iperfTcpTest, ClusterIP: false, MSS: mssMin},
		{SourceNode: "netperf-w3", DestinationNode: "netperf-w2", Label: "4 iperf TCP. Remote VM using Virtual IP", Type: iperfTcpTest, ClusterIP: true, MSS: mssMin},

		{SourceNode: "netperf-w2", DestinationNode: "netperf-w2", Label: "5 iperf TCP. Hairpin Pod to own Virtual IP", Type: iperfTcpTest, ClusterIP: true, MSS: mssMin},

		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "6 iperf UDP. Same VM using Pod IP", Type: iperfUdpTest, ClusterIP: false, MSS: mssMax},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "7 iperf UDP. Same VM using Virtual IP", Type: iperfUdpTest, ClusterIP: true, MSS: mssMax},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w3", Label: "8 iperf UDP. Remote VM using Pod IP", Type: iperfUdpTest, ClusterIP: false, MSS: mssMax},
		{SourceNode: "netperf-w3", DestinationNode: "netperf-w2", Label: "9 iperf UDP. Remote VM using Virtual IP", Type: iperfUdpTest, ClusterIP: true, MSS: mssMax},

		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "10 netperf. Same VM using Pod IP", Type: netperfTest, ClusterIP: false},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w2", Label: "11 netperf. Same VM using Virtual IP", Type: netperfTest, ClusterIP: true},
		{SourceNode: "netperf-w1", DestinationNode: "netperf-w3", Label: "12 netperf. Remote VM using Pod IP", Type: netperfTest, ClusterIP: false},
		{SourceNode: "netperf-w3", DestinationNode: "netperf-w2", Label: "13 netperf. Remote VM using Virtual IP", Type: netperfTest, ClusterIP: true},
	}

	initializeOutputFiles(outputCaptureFile)
	initializeOutputFiles(resultCaptureFile)
	serveRPCRequests(rpcServicePort)
}

func initializeOutputFiles(file string) {
	fd, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		integration.PrettyPrintErr("Failed to open output capture file: %s", err)
		os.Exit(2)
	}
	fd.Close()
}

func serveRPCRequests(port string) {
	baseObject := new(NetPerfRpc)
	rpc.Register(baseObject)
	rpc.HandleHTTP()
	listener, e := net.Listen("tcp", ":"+port)
	if e != nil {
		integration.PrettyPrintErr("rpc listen error: %s", e)
		os.Exit(1)
	}
	http.Serve(listener, nil)
}

// RegisterClient registers a single and assign a work item to it
func (t *NetPerfRpc) RegisterClient(data *types.Worker, reply *types.WorkItem) error {
	globalLock.Lock()
	defer globalLock.Unlock()

	state, ok := workerStateMap[data.Worker]

	if !ok {
		// For new clients, trigger an iperf server start immediately
		state = &types.WorkerState{SentServerItem: true, Idle: true, IP: data.IP, Worker: data.Worker}
		integration.PrettyPrintOk("Registering new client: %+v", state)
		workerStateMap[data.Worker] = state
		reply.IsServerItem = true
		reply.ServerItem.ListenPort = iperf3ServerPort
		reply.ServerItem.Timeout = 3600
		return nil
	} else {
		if debug {
			integration.PrettyPrintSkipped("Client %s already registered", state.Worker)
		}
	}

	// Worker defaults to idle unless the allocateWork routine below assigns an item
	state.Idle = true

	// Give the worker a new work item or let it idle loop another 5 seconds
	allocateWorkToClient(state, reply)
	return nil
}

// ReceiveOutput processes a data received from a single client
func (t *NetPerfRpc) ReceiveOutput(data *types.WorkerOutput, reply *int) error {
	globalLock.Lock()
	defer globalLock.Unlock()

	testcase := testcases[currentJobIndex]

	var outputLog string
	var bw string

	switch data.Type {
	case iperfTcpTest:
		mss := testcases[currentJobIndex].MSS - mssStepSize
		outputLog = outputLog + fmt.Sprintln("Received TCP output from worker", data.Worker, "for test", testcase.Label,
			"from", testcase.SourceNode, "to", testcase.DestinationNode, "MSS:", mss) + data.Output
		writeOutputFile(outputCaptureFile, outputLog)
		bw = parseIperfTcpBandwidth(data.Output)
		registerDataPoint(testcase.Label, mss, bw, currentJobIndex)

	case iperfUdpTest:
		mss := testcases[currentJobIndex].MSS - mssStepSize
		outputLog = outputLog + fmt.Sprintln("Received UDP output from worker", data.Worker, "for test", testcase.Label,
			"from", testcase.SourceNode, "to", testcase.DestinationNode, "MSS:", mss) + data.Output
		writeOutputFile(outputCaptureFile, outputLog)
		bw = parseIperfUdpBandwidth(data.Output)
		registerDataPoint(testcase.Label, mss, bw, currentJobIndex)

	case netperfTest:
		outputLog = outputLog + fmt.Sprintln("Received netperf output from worker", data.Worker, "for test", testcase.Label,
			"from", testcase.SourceNode, "to", testcase.DestinationNode) + data.Output
		writeOutputFile(outputCaptureFile, outputLog)
		bw = parseNetperfBandwidth(data.Output)
		registerDataPoint(testcase.Label, 0, bw, currentJobIndex)
		testcases[currentJobIndex].Finished = true

	}
	integration.PrettyPrintInfo("Job done from worker %s Bandwidth was %s Mbits/sec", data.Worker, bw)
	return nil
}

func allocateWorkToClient(worker *types.WorkerState, reply *types.WorkItem) {
	if !allWorkersIdle() {
		reply.IsIdle = true
		return
	}

	if debug {
		integration.PrettyPrintDebug("System is all idle - pick up next work item to allocate to client")
	}
	for n, v := range testcases {
		if debug {
			integration.PrettyPrintDebug("System is all idle - pick up next work item to allocate to client")
		}
		if v.Finished {
			continue
		}
		if v.SourceNode != worker.Worker {
			reply.IsIdle = true
			return
		}
		if _, ok := workerStateMap[v.DestinationNode]; !ok {
			reply.IsIdle = true
			return
		}
		integration.PrettyPrintInfo("Requesting job '%s' from %s to %s for MSS %d", v.Label, v.SourceNode, v.DestinationNode, v.MSS)
		reply.ClientItem.Type = v.Type
		reply.IsClientItem = true
		worker.Idle = false
		currentJobIndex = n

		if v.ClusterIP {
			reply.ClientItem.Host = getWorkerPodName(v.DestinationNode)
		} else {
			reply.ClientItem.Host = getWorkerPodIP(v.DestinationNode)
		}

		switch {
		case v.Type == iperfTcpTest || v.Type == iperfUdpTest:
			reply.ClientItem.Port = iperf3ServerPort
			reply.ClientItem.MSS = v.MSS

			v.MSS = v.MSS + mssStepSize
			if v.MSS > mssMax {
				v.Finished = true
			}
			return

		case v.Type == netperfTest:
			reply.ClientItem.Port = netperfServerPort
			return
		}
	}

	for _, v := range testcases {
		if !v.Finished {
			return
		}
	}

	if !datapointsFlushed {
		integration.PrettyPrint("ALL TESTCASES AND MSS RANGES COMPLETE - " + csvDataMarker)
		flushDataPointsToCsv()
		datapointsFlushed = true
	}

	reply.IsIdle = true
}

func flushDataPointsToCsv() {
	var buffer string

	// Write the MSS points for the X-axis before dumping all the testcase datapoints
	for _, points := range dataPoints {
		if len(points) == 1 {
			continue
		}
		buffer = fmt.Sprintf("%-45s%s Maximum%s", "MSS", csvSeparator, csvSeparator)
		for _, p := range points {
			buffer = buffer + fmt.Sprintf(" %d%s", p.Mss, csvSeparator)
		}
		break
	}
	integration.PrettyPrint(buffer)

	resultsBuffer := fmt.Sprintf("%s\n", buffer)
	for _, label := range dataPointKeys {
		buffer = fmt.Sprintf("%-45s%s", label, csvSeparator)
		points := dataPoints[label]
		buffer = buffer + fmt.Sprintf("%f%s", getMax(points), csvSeparator)
		for _, p := range points {
			buffer = buffer + fmt.Sprintf("%s%s", p.Bandwidth, csvSeparator)
		}
		integration.PrettyPrint(buffer)
		resultsBuffer += fmt.Sprintf("%s\n", buffer)
	}

	integration.PrettyPrint(csvEndDataMarker)
	resultsBuffer += fmt.Sprintf("%s\n", csvEndDataMarker)
	writeOutputFile(resultCaptureFile, resultsBuffer)
}

func getMax(points []types.Point) float64 {
	var max float64
	for _, p := range points {
		fv, _ := strconv.ParseFloat(p.Bandwidth, 64)
		if fv > max {
			max = fv
		}
	}

	return max
}

func writeOutputFile(filename, data string) {
	fd, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		integration.PrettyPrintWarn("Failed to open existing file %s: %s", filename, err)
		return
	}
	defer fd.Close()

	if _, err = fd.WriteString(data); err != nil {
		integration.PrettyPrintWarn("Failed to append to existing file %s: %s", filename, err)
	}
}

func allWorkersIdle() bool {
	for _, v := range workerStateMap {
		if !v.Idle {
			if debug {
				integration.PrettyPrintDebug("Client %s is not in idle state", v.Worker)
			}
			return false
		}
	}
	return true
}

func getWorkerPodName(worker string) string {
	return workerStateMap[worker].Worker
}

func getWorkerPodIP(worker string) string {
	return workerStateMap[worker].IP
}

func registerDataPoint(label string, mss int, value string, index int) {
	if sl, ok := dataPoints[label]; !ok {
		dataPoints[label] = []types.Point{{Mss: mss, Bandwidth: value, Index: index}}
		dataPointKeys = append(dataPointKeys, label)
	} else {
		dataPoints[label] = append(sl, types.Point{Mss: mss, Bandwidth: value, Index: index})
	}
}

func parseIperfTcpBandwidth(output string) string {
	// Parses the output of iperf3 and grabs the group Mbits/sec from the output
	match := iperfTCPOutputRegexp.FindStringSubmatch(output)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return defaultBandwithFailed
}

func parseIperfUdpBandwidth(output string) string {
	// Parses the output of iperf3 (UDP mode) and grabs the Mbits/sec from the output
	match := iperfUDPOutputRegexp.FindStringSubmatch(output)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return defaultBandwithFailed
}

func parseNetperfBandwidth(output string) string {
	// Parses the output of netperf and grabs the Bbits/sec from the output
	match := netperfOutputRegexp.FindStringSubmatch(output)
	if match != nil && len(match) > 1 {
		return match[1]
	}
	return defaultBandwithFailed
}
