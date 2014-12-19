package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/socketplane/socketplane/ovs"
)

const API_VERSION string = "/v0.1"

type Response struct {
	Status  string
	Message string
}

type Configuration struct {
	BridgeIP   string `json:"bridge_ip"`
	BridgeName string `json:"bridge_name"`
	BridgeCIDR string `json:"bridge_cidr"`
	BridgeMTU  int    `json:"bridge_mtu"`
}

type Connection struct {
	ContainerID       string            `json:"container_id"`
	ContainerName     string            `json:"container_name"`
	ContainerPID      string            `json:"container_pid"`
	OvsPortID         string            `json:"ovs_port_id"`
	ConnectionDetails ovs.OvsConnection `json:"connection_details"`
}

type apiError struct {
	Code    int
	Message string
}

type HttpApiFunc func(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError

type appHandler struct {
	*Daemon
	h HttpApiFunc
}

func (ah appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := ah.h(ah.Daemon, w, r)
	if err != nil {
		http.Error(w, err.Message, err.Code)
	}
}

func ServeAPI(d *Daemon) {
	r := createRouter(d)
	server := &http.Server{
		Addr:    "0.0.0.0:6675",
		Handler: r,
	}
	server.ListenAndServe()
}

func createRouter(d *Daemon) *mux.Router {
	r := mux.NewRouter()
	m := map[string]map[string]HttpApiFunc{
		"GET": {
			"/configuration":       getConfiguration,
			"/connections":         getConnections,
			"/connections/{id:.*}": getConnection,
		},
		"POST": {
			"/configuration": setConfiguration,
			"/connections":   createConnection,
		},
		"DELETE": {
			"/connections/{id:.*}": deleteConnection,
		},
	}

	for method, routes := range m {
		for route, fct := range routes {
			handler := appHandler{d, fct}
			r.Path(API_VERSION + route).Methods(method).Handler(handler)
		}
	}

	return r
}

func getConfiguration(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	data, _ := json.Marshal(d.Configuration)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
	return nil
}

func setConfiguration(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	if r.Body == nil {
		return &apiError{http.StatusBadRequest, "Request body is empty"}
	}
	cfg := &Configuration{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(cfg)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	d.Configuration = cfg
	return nil
}

func getConnections(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	data, _ := json.Marshal(d.Connections)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
	return nil
}

func getConnection(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	vars := mux.Vars(r)
	containerID := vars["id"]
	connection := d.Connections[containerID]

	if connection == nil {
		msg := fmt.Sprintf("Connection for container %v not found", containerID)
		return &apiError{http.StatusNotFound, msg}
	}

	data, _ := json.Marshal(connection)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
	return nil
}

func createConnection(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	if r.Body == nil {
		return &apiError{http.StatusBadRequest, "Request body is empty"}
	}
	cfg := &Connection{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(cfg)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	pid, err := strconv.Atoi(cfg.ContainerPID)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	ovsConnection, err := ovs.AddConnection(pid)
	if err != nil && ovsConnection.Name == "" {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	cfg.OvsPortID = ovsConnection.Name
	cfg.ConnectionDetails = ovsConnection
	d.Connections[cfg.ContainerID] = cfg

	data, _ := json.Marshal(cfg)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
	return nil
}

func deleteConnection(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	vars := mux.Vars(r)
	containerID := vars["id"]

	portID := d.Connections[containerID].OvsPortID
	err := ovs.DeleteConnection(portID)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	return nil
}
