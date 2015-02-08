package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestGetConfigurationEmpty(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("GET", "/v0.1/configuration", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	fmt.Println(response.Body)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestGetConfiguration(t *testing.T) {
	daemon := NewDaemon()
	daemon.Configuration = &Configuration{
		BridgeIP:   "172.16.42.1",
		BridgeName: "socketplane0",
		BridgeCIDR: "172.16.42.0/24",
		BridgeMTU:  1460,
	}
	request, _ := http.NewRequest("GET", "/v0.1/configuration", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	fmt.Println(response.Body)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestSetConfigurationNoBody(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/configuration", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("Expected %v:\n\tReceived: %v", "400", response.Code)
	}
}

func TestSetConfigurationBadBody(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/configuration", bytes.NewReader([]byte{1, 2, 3, 4}))
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("Expected %v:\n\tReceived: %v", "500", response.Code)
	}
}

func TestSetConfiguration(t *testing.T) {
	daemon := NewDaemon()
	cfg := &Configuration{
		BridgeIP:   "172.16.42.1",
		BridgeName: "socketplane0",
		BridgeCIDR: "172.16.42.0/24",
		BridgeMTU:  1460,
	}
	data, _ := json.Marshal(cfg)
	request, _ := http.NewRequest("POST", "/v0.1/configuration", bytes.NewReader(data))
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestGetConnections(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("GET", "/v0.1/connections", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestGetConnection(t *testing.T) {
	daemon := NewDaemon()
	connection := &Connection{
		ContainerID:   "abc123",
		ContainerName: "test_container",
		ContainerPID:  "1234",
		Network:       "default",
	}
	daemon.Connections["abc123"] = connection
	request, _ := http.NewRequest("GET", "/v0.1/connections/abc123", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "404", response.Code)
	}

	expected, _ := json.Marshal(connection)
	if !bytes.Equal(response.Body.Bytes(), expected) {
		t.Fatal("body does not match")
	}

	headers := response.HeaderMap["Content-Type"]
	if headers[0] != "application/json; charset=utf-8" {
		t.Fatal("headers not correctly set")
	}
}

func TestCreateConnection(t *testing.T) {
	daemon := NewDaemon()
	connection := &Connection{
		ContainerID:   "abc123",
		ContainerName: "test_container",
		ContainerPID:  "1234",
		Network:       "foo",
	}
	data, _ := json.Marshal(connection)
	request, _ := http.NewRequest("POST", "/v0.1/connections", bytes.NewReader(data))
	response := httptest.NewRecorder()

	go func() {
		for {
			context := <-daemon.cC
			if context == nil {
				t.Fatalf("Object taken from channel is nil")
			}
			if context.Action != ConnectionAdd {
				t.Fatal("should be adding a new connection")
			}

			if !reflect.DeepEqual(context.Connection, connection) {
				t.Fatal("payload is incorrect")
			}
			context.Result <- connection
		}
	}()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v:\n\t%v", "200", response.Code, response.Body)
	}

	if !bytes.Equal(data, response.Body.Bytes()) {
		t.Fatalf("body is not correct")
	}

	contentHeader := response.HeaderMap["Content-Type"]
	if contentHeader[0] != "application/json; charset=utf-8" {
		t.Fatal("headers not correctly set")
	}

	locationHeader := response.HeaderMap["Content-Location"]
	fmt.Println(locationHeader[0])
	expected := "/v0.1/connections/abc123"
	if locationHeader[0] != expected {
		t.Fatal("header not correctly set")
	}
}

func TestCreateConnectionNoNetwork(t *testing.T) {
	daemon := NewDaemon()
	connection := &Connection{
		ContainerID:   "abc123",
		ContainerName: "test_container",
		ContainerPID:  "1234",
		Network:       "",
	}
	expected := &Connection{
		ContainerID:   "abc123",
		ContainerName: "test_container",
		ContainerPID:  "1234",
		Network:       "default",
	}

	data, _ := json.Marshal(connection)
	request, _ := http.NewRequest("POST", "/v0.1/connections", bytes.NewReader(data))
	response := httptest.NewRecorder()

	go func() {
		for {
			context := <-daemon.cC
			if context == nil {
				t.Fatalf("Object taken from channel is nil")
			}
			if context.Action != ConnectionAdd {
				t.Fatal("should be adding a new connection")
			}

			if !reflect.DeepEqual(context.Connection, expected) {
				t.Fatal("payload is incorrect")
			}
			context.Result <- expected
		}
	}()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v:\n\t%v", "200", response.Code, response.Body)
	}

	expectedBody, _ := json.Marshal(expected)
	if !bytes.Equal(expectedBody, response.Body.Bytes()) {
		t.Fatalf("body is not correct")
	}

	contentHeader := response.HeaderMap["Content-Type"]
	if contentHeader[0] != "application/json; charset=utf-8" {
		t.Fatal("headers not correctly set")
	}

	locationHeader := response.HeaderMap["Content-Location"]
	fmt.Println(locationHeader[0])
	expectedHeader := "/v0.1/connections/abc123"
	if locationHeader[0] != expectedHeader {
		t.Fatal("header not correctly set")
	}
}

func TestCreateConnectionNoBody(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/connections", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("Expected %v:\n\tReceived: %v", "400", response.Code)
	}
}

func TestCreateConnectionBadBody(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/connections", bytes.NewReader([]byte{1, 2, 3, 4}))
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("Expected %v:\n\tReceived: %v", "500", response.Code)
	}
}

func TestGetConnectionNonExistent(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("GET", "/v0.1/connections/abc123", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("Expected %v:\n\tReceived: %v", "404", response.Code)
	}
}

func TestDeleteConnectionNonExistent(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("DELETE", "/v0.1/connections/abc123", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("Expected %v:\n\tReceived: %v", "404", response.Code)
	}
}

func TestDeleteConnection(t *testing.T) {
	daemon := NewDaemon()
	connection := &Connection{
		ContainerID:   "abc123",
		ContainerName: "test_container",
		ContainerPID:  "1234",
		Network:       "default",
	}
	daemon.Connections["abc123"] = connection
	request, _ := http.NewRequest("DELETE", "/v0.1/connections/abc123", nil)
	response := httptest.NewRecorder()

	go func() {
		for {
			context := <-daemon.cC
			if context == nil {
				t.Fatalf("Object taken from channel is nil")
			}
			if context.Action != ConnectionDelete {
				t.Fatal("should be adding a new connection")
			}

			if !reflect.DeepEqual(context.Connection, connection) {
				t.Fatal("payload is incorrect")
			}
			context.Result <- connection
		}
	}()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}

}

func TestGetNetworksApi(t *testing.T) {
	t.Skip("unable to mock network store")
	daemon := NewDaemon()
	request, _ := http.NewRequest("GET", "/v0.1/networks", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestGetNetworkApi(t *testing.T) {
	t.Skip("unable to mock network store")
	daemon := NewDaemon()
	/* ToDo: How do we inject this network?
	network := &Network{
		ID:      "foo",
		Subnet:  "10.10.10.0/24",
		Gateway: "10.10.10.1",
		Vlan:    uint(1),
	}
	*/

	request, _ := http.NewRequest("GET", "/v0.1/networks/foo", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestSetNetworksApi(t *testing.T) {
	t.Skip("unable to mock network store")
	daemon := NewDaemon()
	network := &Network{
		ID:      "foo",
		Subnet:  "10.10.10.0/24",
		Gateway: "10.10.10.1",
		Vlan:    uint(1),
	}
	data, _ := json.Marshal(network)

	request, _ := http.NewRequest("POST", "/v0.1/networks", bytes.NewReader(data))
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestDeleteNetworksApi(t *testing.T) {
	t.Skip("unable to mock network store")
	daemon := NewDaemon()
	/* ToDo: How do we inject this network?
	network := &Network{
		ID:      "foo",
		Subnet:  "10.10.10.0/24",
		Gateway: "10.10.10.1",
		Vlan:    uint(1),
	}
	*/

	request, _ := http.NewRequest("DELETE", "/v0.1/networks", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestGetNetworkNonExistentApi(t *testing.T) {
	t.Skip("unable to mock network store")
	daemon := NewDaemon()
	request, _ := http.NewRequest("GET", "/v0.1/networks/abc123", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusNotFound {
		t.Fatalf("Expected %v:\n\tReceived: %v", "404", response.Code)
	}
}

func TestDeleteNetworkNonExistentApi(t *testing.T) {
	t.Skip("unable to mock network store")
	daemon := NewDaemon()
	request, _ := http.NewRequest("DELETE", "/v0.1/connections/abc123", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("Expected %v:\n\tReceived: %v", "404", response.Code)
	}
}

func TestClusterJoin(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/join?address=1.1.1.1", nil)
	response := httptest.NewRecorder()

	go createRouter(daemon).ServeHTTP(response, request)
	foo := <-daemon.bindChan
	if foo == nil {
		t.Fatal("object from bindChan is nil")
	}

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestClusterJoiniBadIp(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/join?address=bar", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("Expected %v:\n\tReceived: %v", "500", response.Code)
	}
}

func TestClusterJoinNoParams(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/join", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatal("request should fail")
	}
}

func TestClusterJoinBadParams(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/join?foo!@£%£", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatal("request should fail")
	}
}

func TestClusterJoinBadParams2(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/join?foo=bar", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatal("request should fail")
	}
}

func TestClusterLeave(t *testing.T) {
	t.Skip("Not implemented")
}

func TestClusterBind(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/bind?iface=eth0", nil)
	response := httptest.NewRecorder()

	go createRouter(daemon).ServeHTTP(response, request)
	foo := <-daemon.bindChan
	if foo == nil {
		t.Fatal("object from bindChan is nil")
	}

	if response.Code != http.StatusOK {
		t.Fatalf("Expected %v:\n\tReceived: %v", "200", response.Code)
	}
}

func TestClusterBindBadIface(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/bind?iface=foo123", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("Expected %v:\n\tReceived: %v", "500", response.Code)
	}
}

func TestClusterBindNoParams(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/bind", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatal("request should fail")
	}
}

func TestClusterBindBadParams(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/bind?foo!@£%£", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatal("request should fail")
	}
}

func TestClusterBindBadParams2(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/cluster/bind?foo=bar", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatal("request should fail")
	}
}
