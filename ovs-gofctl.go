
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

func usage() {
    fmt.Println("Usage : ovs-gofctl <bridge-name>")
}

func main() {
    if len(os.Args) < 2 {
        usage()
        return
    }
    fmt.Printf("Show returns : %s\n", C.GoString(C.show_ofctl(C.CString(os.Args[1]))));
}
