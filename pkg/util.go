package pkg

var debug bool

// Worker specific
const (
	WorkerMode        = "worker"
	iperf3Path        = "/usr/bin/iperf3"
	netperfPath       = "/usr/local/bin/netperf"
	netperfServerPath = "/usr/local/bin/netserver"
	parallelStreams   = "8"

	EnvOrchestratorPort  = "orchestratorPort"
	EnvOrchestratorPodIP = "orchestratorPodIP"
	EnvWorkerPodIP       = "workerPodIP"
	EnvWorkerName        = "workerName"
)

// Orchestrator specific
const (
	OrchestratorMode  = "orchestrator"
	outputCaptureFile = "/tmp/output.txt"
	resultCaptureFile = "/tmp/result.csv"
	mssMin            = 96
	mssMax            = 1460
	mssStepSize       = 64

	rpcServicePort    = "5202"
	iperf3ServerPort  = "5201"
	netperfServerPort = "12865"

	csvDataMarker    = "GENERATING CSV OUTPUT"
	csvEndDataMarker = "END CSV DATA"
)

const (
	iperfTcpTest = iota
	iperfUdpTest = iota
	netperfTest  = iota
)
