package daemon

import (
	"fmt"
	"os"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/libovsdb"
)

func ovsTestFunc() {
	// ToDo: Something here. For now, just keeping the libovsdb dependency satisfied

	// By default libovsdb connects to 127.0.0.0:6400.
	ovs, err := libovsdb.Connect("", 0)

	// If you prefer to connect to OVS in a specific location :
	// ovs, err := libovsdb.Connect("192.168.56.101", 6640)
	if err != nil {
		fmt.Println("Unable to Connect ", err)
		os.Exit(1)
	}

	ovs.Disconnect()
}
