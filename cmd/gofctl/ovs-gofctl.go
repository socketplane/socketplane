// ovs-gofctl is a sample Go program that make use of cgo to access
// OVS libraries and executes functions that is equivalent to
// 'ovs-ofctl show' command

package main

// #cgo CFLAGS: -I./third-party/ovs-driver/include
// #cgo LDFLAGS: -L./third-party/ovs-driver/lib -lovsdriver -lopenvswitch -lofproto -lrt -lm
// #include <ovs-driver.h>
import "C"
import (
	"fmt"
	"os"
)

func show_usage() {
	fmt.Println("Usage : ovs-gofctl <command> <bridge-name>")
}

func flow_mod_usage() {
	fmt.Println("Usage : ovs-gofctl <command> <bridge-name> <flow>")
}

func main() {
	switch os.Args[1] {
	case "show":
		if len(os.Args) < 3 {
			show_usage()
			return
		}
		fmt.Printf("Show returns : %s\n",
			C.GoString(
				C.ovs_get_features(
					C.CString(os.Args[2]))))
	case "add-flow":
		if len(os.Args) < 4 {
			flow_mod_usage()
			return
		}
                C.ovs_add_flow(C.CString(os.Args[2]), C.CString(os.Args[3]))
	case "del-flow":
		if len(os.Args) < 4 {
			flow_mod_usage()
			return
		}
                C.ovs_del_flow(C.CString(os.Args[2]), C.CString(os.Args[3]))
	case "mod-flow":
		if len(os.Args) < 4 {
			flow_mod_usage()
			return
		}
                C.ovs_mod_flow(C.CString(os.Args[2]), C.CString(os.Args[3]))
	default:
	}
}
