package integration

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

const (
	noType      = ""
	okType      = "[OK]"
	errType     = "[ERROR]"
	skippedType = "[SKIPPED]"
	warnType    = "[WARNING]"
	unknownType = "[UNKNOWN]"
	ignoredType = "[IGNORED]"
	infoType    = "[INFO]"
	debugType   = "[DEBUG]"
)

var out io.Writer = os.Stdout

// PrettyPrintOk [OK] with formatted string
func PrettyPrintOk(msg string, a ...interface{}) {
	printMsg(msg, okType, a...)
}

// PrettyPrintErr [ERROR] with formatted string
func PrettyPrintErr(msg string, a ...interface{}) {
	printMsg(msg, errType, a...)
}

// PrettyPrint no type will be displayed, used for just single line printing
func PrettyPrint(msg string, a ...interface{}) {
	printMsg(msg, noType, a...)
}

// PrettyPrintWarn [WARNING] with formatted string
func PrettyPrintWarn(msg string, a ...interface{}) {
	printMsg(msg, warnType, a...)
}

// PrettyPrintIgnored [IGNORED] with formatted string
func PrettyPrintIgnored(msg string, a ...interface{}) {
	printMsg(msg, ignoredType, a...)
}

// PrettyPrintInfo [INFO] with formatted string
func PrettyPrintInfo(msg string, a ...interface{}) {
	printMsg(msg, infoType, a...)
}

// PrettyPrintDebug [DEBUG] with formatted string
func PrettyPrintDebug(msg string, a ...interface{}) {
	printMsg(msg, debugType, a...)
}

// PrettyPrintUnknown [UNREACHABLE] with formatted string
func PrettyPrintUnknown(msg string, a ...interface{}) {
	printMsg(msg, unknownType, a...)
}

// PrettyPrintSkipped [SKIPPED] with formatted string
func PrettyPrintSkipped(msg string, a ...interface{}) {
	printMsg(msg, skippedType, a...)
}

// PrintHeader will print header with predefined width
func PrintHeader(msg string, padding byte) {
	w := tabwriter.NewWriter(out, 104, 0, 0, padding, 0)
	fmt.Fprintln(w, "")
	format := msg + "\t\n"
	fmt.Fprintf(w, format)
	w.Flush()
}

func printMsg(msg, status string, a ...interface{}) {
	w := tabwriter.NewWriter(out, 100, 0, 0, ' ', 0)
	// print message
	format := msg + "\t"
	fmt.Fprintf(w, format, a...)

	// print status
	if status != noType {
		fmt.Fprintf(w, "%s\n", status)
	} else {
		fmt.Fprint(w, "\n")
	}

	w.Flush()
}
