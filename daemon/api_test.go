package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Expected %v:\n\tReceived: %v", "501", response.Code)
	}
}

func TestGetConnection(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("GET", "/v0.1/connections/abc123", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Expected %v:\n\tReceived: %v", "501", response.Code)
	}
}

func TestCreateConnection(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("POST", "/v0.1/connections", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Expected %v:\n\tReceived: %v", "501", response.Code)
	}
}

func TestDeleteConnection(t *testing.T) {
	daemon := NewDaemon()
	request, _ := http.NewRequest("DELETE", "/v0.1/connections/abc123", nil)
	response := httptest.NewRecorder()

	createRouter(daemon).ServeHTTP(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Expected %v:\n\tReceived: %v", "501", response.Code)
	}
}
