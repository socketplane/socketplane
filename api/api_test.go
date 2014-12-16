package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetConfiguration(t *testing.T) {

	request, _ := http.NewRequest("GET", "/v0.1/configuration", nil)
	response := httptest.NewRecorder()

	getConfiguration(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "200", response.Code)
	}
}

func TestSetConfiguration(t *testing.T) {

	request, _ := http.NewRequest("POST", "/v0.1/configuration", nil)
	response := httptest.NewRecorder()

	setConfiguration(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "200", response.Code)
	}
}

func TestGetConnections(t *testing.T) {

	request, _ := http.NewRequest("GET", "/v0.1/connections", nil)
	response := httptest.NewRecorder()

	getConnections(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "200", response.Code)
	}
}

func TestGetConnection(t *testing.T) {

	request, _ := http.NewRequest("GET", "/v0.1/connections/abc123", nil)
	response := httptest.NewRecorder()

	getConnection(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "200", response.Code)
	}
}

func TestCreateConnection(t *testing.T) {

	request, _ := http.NewRequest("POST", "/v0.1/connections", nil)
	response := httptest.NewRecorder()

	createConnection(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "200", response.Code)
	}
}

func TestDeleteConnection(t *testing.T) {

	request, _ := http.NewRequest("DELETE", "/v0.1/connections/abc123", nil)
	response := httptest.NewRecorder()

	deleteConnection(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "200", response.Code)
	}
}
