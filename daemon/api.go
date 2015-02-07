package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/gorilla/mux"
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
	ContainerID       string        `json:"container_id"`
	ContainerName     string        `json:"container_name"`
	ContainerPID      string        `json:"container_pid"`
	Network           string        `json:"network"`
	OvsPortID         string        `json:"ovs_port_id"`
	ConnectionDetails OvsConnection `json:"connection_details"`
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
			"/networks":            getNetworks,
			"/networks/{id:.*}":    getNetwork,
		},
		"POST": {
			"/configuration": setConfiguration,
			"/connections":   createConnection,
			"/networks":      createNetwork,
			"/cluster/bind":  clusterBind,
			"/cluster/join":  clusterJoin,
			"/cluster/leave": clusterLeave,
			"/adapter":       psAdapter,
		},
		"DELETE": {
			"/connections/{id:.*}": deleteConnection,
			"/networks/{id:.*}":    deleteNetwork,
		},
	}

	for method, routes := range m {
		for route, fct := range routes {
			handler := appHandler{d, fct}
			r.Path(API_VERSION + route).Methods(method).Handler(handler)
			if route == "/adapter" {
				r.Path(route).Methods(method).Handler(handler)
			}
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

	if cfg.Network == "" {
		cfg.Network = DefaultNetworkName
	}

	context := &ConnectionContext{
		ConnectionAdd,
		cfg,
		make(chan *Connection),
	}
	d.cC <- context

	result := <-context.Result

	location := fmt.Sprintf("%S/%s", r.URL.String(), cfg.ContainerID)
	data, _ := json.Marshal(result)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Location", location)
	w.Write(data)
	return nil
}

func deleteConnection(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	vars := mux.Vars(r)
	containerID := vars["id"]

	connection, ok := d.Connections[containerID]
	if !ok {
		return &apiError{http.StatusNotFound, "Container Not Found"}
	}

	context := &ConnectionContext{
		ConnectionDelete,
		connection,
		make(chan *Connection),
	}
	d.cC <- context
	return nil
}

func getNetworks(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	networks, err := GetNetworks()
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	data, err := json.Marshal(networks)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
	return nil
}

func getNetwork(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	vars := mux.Vars(r)
	networkID := vars["id"]

	networks, err := GetNetwork(networkID)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	data, err := json.Marshal(networks)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
	return nil
}

func createNetwork(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	if r.Body == nil {
		return &apiError{http.StatusBadRequest, "Request body is empty"}
	}
	networkRequest := &Network{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(networkRequest)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	_, cidr, err := net.ParseCIDR(networkRequest.Subnet)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}

	newNetwork, err := CreateNetwork(networkRequest.ID, cidr)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}

	data, _ := json.Marshal(newNetwork)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
	return nil
}

func deleteNetwork(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	vars := mux.Vars(r)
	networkID := vars["id"]

	err := DeleteNetwork(networkID)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	return nil
}

func clusterBind(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	if r.URL.RawQuery == "" {
		return &apiError{http.StatusBadRequest, "Please provide the interface parameter"}
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return &apiError{http.StatusBadRequest, "Could not decode query parameters"}
	}
	iface, ok := values["iface"]
	if !ok || iface[0] == "" {
		return &apiError{http.StatusBadRequest, "Please provide the interface parameter"}
	}
	log.Debugf("Request Received. Change Cluster Interface to %s", iface[0])
	err = d.ConfigureClusterListenerPort(iface[0])
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	return nil
}

func clusterJoin(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	if r.URL.RawQuery == "" {
		return &apiError{http.StatusBadRequest, "Please provide the address parameter"}
	}
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return &apiError{http.StatusBadRequest, "Could not decode query parameters"}
	}
	addr, ok := values["address"]
	if !ok || addr[0] == "" {
		return &apiError{http.StatusBadRequest, "Please provide the address parameter"}
	}
	log.Debugf("Request Received. Join Cluster  %s", addr[0])
	err = d.JoinCluster(addr[0])
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	return nil
}

func clusterLeave(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	//ToDo: Needs implementing
	return nil
}
