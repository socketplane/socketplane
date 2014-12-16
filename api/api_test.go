package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetConnections(t *testing.T) {

	request, _ := http.NewRequest("GET", "/v0.1/connections", nil)
	response := httptest.NewRecorder()

	getConnection(response, request)

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

	getConnection(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "200", response.Code)
	}
}

func TestDeleteConnection(t *testing.T) {

	request, _ := http.NewRequest("DELETE", "/v0.1/connections", nil)
	response := httptest.NewRecorder()

	getConnection(response, request)

	if response.Code != http.StatusNotImplemented {
		t.Fatalf("Non-expected status code%v:\n\tbody: %v", "200", response.Code)
	}
}
