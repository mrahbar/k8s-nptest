package pkg

import (
	"bytes"
	"github.com/mrahbar/k8s-nptest/integration"
	"github.com/mrahbar/k8s-nptest/types"
	"net/rpc"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var orchestrator types.Orchestrator
var clientData types.Worker

//Visit sites for iperf and netperf args documentation
// http://software.es.net/iperf/invoking.html
// http://www.cs.kent.edu/~farrell/dist/ref/Netperf.html
func Work(d bool) {
	debug = d

	orchestrator.Port = os.Getenv(EnvOrchestratorPort)
	orchestrator.Address = os.Getenv(EnvOrchestratorPodIP)
	clientData.IP = os.Getenv(EnvWorkerPodIP)
	clientData.Worker = os.Getenv(EnvWorkerName)

	startWork()
}

// Entry point to the worker infinite loop
func startWork() {
	for true {
		var timeout time.Duration
		var client *rpc.Client
		var err error

		timeout = 5
		for true {
			integration.PrettyPrintInfo("Attempting to connect to orchestrator at %s", orchestrator.Address)
			client, err = rpc.DialHTTP("tcp", orchestrator.Address+":"+orchestrator.Port)
			if err == nil {
				integration.PrettyPrintOk("Connected successfully to orchestrator")
				break
			}
			integration.PrettyPrintWarn("RPC connection to %s on port %s failed: %s", orchestrator.Address, orchestrator.Port, err)
			time.Sleep(timeout * time.Second)
		}

		for true {
			var workItem types.WorkItem

			if err := client.Call("NetPerfRpc.RegisterClient", clientData, &workItem); err != nil {
				// RPC server has probably gone away - attempt to reconnect
				integration.PrettyPrintErr("Error attempting RPC call: %s", err)
				break
			}

			switch {
			case workItem.IsIdle == true:
				if debug {
					integration.PrettyPrintInfo("Orchestrator requests worker run idle")
				}
				time.Sleep(5 * time.Second)
				continue

			case workItem.IsServerItem == true:
				integration.PrettyPrintInfo("Orchestrator requests worker run iperf and netperf server")
				go iperfServer(iperf3ServerPort)
				go netperfServer(netperfServerPort)
				time.Sleep(1 * time.Second)

			case workItem.IsClientItem == true:
				integration.PrettyPrintInfo("Orchestrator requests worker run as client: %+v", workItem.ClientItem)
				handleClientWorkItem(client, &workItem)
			}
		}
	}
}

func handleClientWorkItem(client *rpc.Client, workItem *types.WorkItem) {
	var reply int
	switch {
	case workItem.ClientItem.Type == iperfTcpTest || workItem.ClientItem.Type == iperfUdpTest:
		integration.PrettyPrintInfo("Orchestrator requests worker run item Type: iperfTest")
		outputString := iperfClient(workItem.ClientItem.Host, workItem.ClientItem.Port, workItem.ClientItem.MSS, workItem.ClientItem.Type)
		client.Call("NetPerfRpc.ReceiveOutput", types.WorkerOutput{Output: outputString, Worker: clientData.Worker, Type: workItem.ClientItem.Type}, &reply)
	case workItem.ClientItem.Type == netperfTest:
		integration.PrettyPrintInfo("Orchestrator requests worker run item Type: netperfTest")
		outputString := netperfClient(workItem.ClientItem.Host, workItem.ClientItem.Port)
		client.Call("NetPerfRpc.ReceiveOutput", types.WorkerOutput{Output: outputString, Worker: clientData.Worker, Type: workItem.ClientItem.Type}, &reply)
	}
	// Client COOLDOWN period before asking for next work item to replenish burst allowance polices etc
	time.Sleep(10 * time.Second)
}

// Invoke and indefinitely run an iperf server
func iperfServer(port string) {
	integration.PrettyPrintInfo("Starting iperf server on %s", clientData.Worker)
	args := []string{"-s", "-p", port}
	if debug {
		args = append(args, "-V", "-d")
	}
	output, success := cmdExec(iperf3Path, args)
	if success {
		integration.PrettyPrintInfo(output)
	}
}

// Invoke and indefinitely run netperf server
func netperfServer(port string) {
	integration.PrettyPrintInfo("Starting netperf server on %s", clientData.Worker)
	args := []string{"-D", "-p", port}
	if debug {
		args = append(args, "-d")
	}
	output, success := cmdExec(netperfServerPath, args)
	if success {
		integration.PrettyPrintInfo(output)
	}
}

// Invoke and run an iperf client and return the output if successful.
func iperfClient(serverHost, serverPort string, mss int, workItemType int) (rv string) {
	switch {
	case workItemType == iperfTcpTest:
		integration.PrettyPrintInfo("Starting iperf tcp client on %s to %s", clientData.Worker, serverHost)
		output, success := cmdExec(iperf3Path, []string{"-c", serverHost, "-p", serverPort, "-N", "-i", "30", "-t", "10", "-f", "m", "-w", "512M", "-Z", "-P", parallelStreams, "-M", strconv.Itoa(mss)})
		if success {
			rv = output
		}

	case workItemType == iperfUdpTest:
		integration.PrettyPrintInfo("Starting iperf udp client on %s to %s", clientData.Worker, serverHost)
		output, success := cmdExec(iperf3Path, []string{"-c", serverHost, "-p", serverPort, "-i", "30", "-t", "10", "-f", "m", "-b", "0", "-u"})
		if success {
			rv = output
		}
	}
	return
}

// Invoke and run a netperf client and return the output if successful.
func netperfClient(serverHost, serverPort string) (rv string) {
	//measures measure bulk tcp data transfer performance
	integration.PrettyPrintInfo("Starting netperf client on %s to %s", clientData.Worker, serverHost)
	output, success := cmdExec(netperfPath, []string{"-H", serverHost, "-p", serverPort})
	if success {
		integration.PrettyPrintInfo(output)
		rv = output
	} else {
		integration.PrettyPrintErr("Error running netperf client %s", output)
	}

	return
}

func cmdExec(binaryPath string, args []string) (rv string, rc bool) {
	if debug {
		integration.PrettyPrintDebug("Calling command: %s %s", binaryPath, strings.Join(args, " "))
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdin = os.Stdin

	var stdoutput bytes.Buffer
	var stderror bytes.Buffer
	cmd.Stdout = &stdoutput
	cmd.Stderr = &stderror

	if err := cmd.Run(); err != nil {
		outputstr := stdoutput.String()
		errstr := stderror.String()
		integration.PrettyPrintErr("Failed to run '%s': Result: %s Error: %s - %s", binaryPath, outputstr, errstr, err)
		return
	}

	rv = stdoutput.String()
	rc = true
	return
}
