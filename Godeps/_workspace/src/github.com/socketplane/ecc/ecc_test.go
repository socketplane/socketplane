package ecc

import (
	"bytes"
	"testing"
)

func TestStart(t *testing.T) {
	err := Start(true, true, "", "data-dir")
	if err != nil {
		t.Error("Error starting Consul ", err)
	}
}

func TestJoin(t *testing.T) {
	err := Join("1.1.1.1")
	if err == nil {
		t.Error("Join to unknown peer must fail")
	}
}

func TestGet(t *testing.T) {
	existingValue, _, ok := Get("ipam", "test")
	if ok {
		t.Fatal("Please cleanup the existing database and restart the test :", string(existingValue[:]))
	}
}

func TestPut(t *testing.T) {
	existingValue, _, ok := Get("ipam", "test")
	if ok {
		t.Fatal("Please cleanup the existing database and restart the test")
	}

	eccerr := Put("ipam", "test", []byte("192.168.56.1"), existingValue)
	if eccerr != OK {
		t.Fatal("Error putting value into ipam store")
	}

	// Test with Old existingValue
	eccerr = Put("ipam", "test", []byte("192.168.56.1"), existingValue)
	if eccerr == OK {
		t.Fatal("Put must fail if the existingValue is NOT in sync with the db")
	}

	// Test with New existingValue
	existingValue, _, ok = Get("ipam", "test")
	if !ok {
		t.Fatal("test key is missing in ipam store")
	}

	eccerr = Put("ipam", "test", []byte("192.168.56.2"), existingValue)
	if eccerr != OK {
		t.Error("Error putting value into ipam store")
	}

	existingValue, _, ok = Get("ipam", "test")
	if !ok {
		t.Fatal("test key is missing in ipam store")
	}
	if !bytes.Equal(existingValue, []byte("192.168.56.2")) {
		t.Fatal("Value for test key is not updated in DB")
	}
}

func TestCleanup(t *testing.T) {
	Delete("ipam", "test")
	Leave()
}

func TestRestart(t *testing.T) {
	err := Start(true, true, "", "data-dir")
	if err != nil {
		t.Error("Error starting Consul ", err)
	}
	Leave()
}
