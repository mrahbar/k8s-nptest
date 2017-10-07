package types

// IperfClientWorkItem represents a single task for an Iperf client
type IperfClientWorkItem struct {
	Host string
	Port string
	MSS  int // TCP/SCTP maximum segment size (MTU - 40 bytes)
	Type int
}

// IperfServerWorkItem represents a single task for an Iperf server
type IperfServerWorkItem struct {
	ListenPort string
	Timeout    int
}

// WorkItem represents a single task for a worker
type WorkItem struct {
	IsClientItem bool
	IsServerItem bool
	IsIdle       bool
	ClientItem   IperfClientWorkItem
	ServerItem   IperfServerWorkItem
}

type WorkerState struct {
	SentServerItem bool
	Idle           bool
	IP             string
	Worker         string
}

// WorkerOutput stores the results from a single worker
type WorkerOutput struct {
	Output string
	Code   int
	Worker string
	Type   int
}

type Testcase struct {
	SourceNode      string
	DestinationNode string
	Label           string
	ClusterIP       bool
	Finished        bool
	MSS             int
	Type            int
}
