package daemon

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/socketplane/socketplane/datastore"
	"github.com/socketplane/socketplane/ovs"
)

const (
	API_VERSION  string = "/v0.1"
	NODE_ADDRESS        = "nodeAddr"
	CLUSTER             = "/cluster"
	BOOSTRAP            = "/boostrap"
    BOOSTRAP_IFACE        = "iface"
)

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
	Network           string            `json:"network"`
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
			"/networks":            getNetworks,
			"/networks/{id:.*}":    getNetwork,
		},
		"POST": {
			"/configuration": setConfiguration,
			"/connections":   createConnection,
			"/networks":      createNetwork,
			CLUSTER + "/{" + NODE_ADDRESS + "}": addClusterHandler,
			CLUSTER + BOOSTRAP + "/{" + BOOSTRAP_IFACE + "}": addBoostrapHandler,
		},
		"DELETE": {
			"/connections/{id:.*}": deleteConnection,
			"/networks/{id:.*}":    deleteNetwork,
			CLUSTER + "/{" + NODE_ADDRESS + "}": delClusterHandler,
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

	if cfg.Network == "" {
		cfg.Network = ovs.DefaultNetworkName
	}
	ovsConnection, err := ovs.AddConnection(pid, cfg.Network)
	if err != nil && ovsConnection.Name == "" {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	cfg.OvsPortID = ovsConnection.Name
	cfg.ConnectionDetails = ovsConnection
	d.Connections[cfg.ContainerID] = cfg

	data, _ := json.Marshal(cfg)
	ovs.UpdateConnectionContext(ovsConnection.Name, cfg.ContainerID, string(data[:]))

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
	return nil
}

func deleteConnection(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	vars := mux.Vars(r)
	containerID := vars["id"]

	connection, ok := d.Connections[containerID]
	if !ok {
		return nil
	}

	err := ovs.DeleteConnection(connection.ConnectionDetails)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	delete(d.Connections, containerID)
	return nil
}

func getNetworks(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	networks, err := ovs.GetNetworks()
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

	networks, err := ovs.GetNetwork(networkID)
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
	networkRequest := &ovs.Network{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(networkRequest)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	_, cidr, err := net.ParseCIDR(networkRequest.Subnet)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}

	newNetwork, err := ovs.CreateNetwork(networkRequest.ID, cidr)
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

	err := ovs.DeleteNetwork(networkID)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	return nil
}

func addClusterHandler(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	vars := mux.Vars(r)
	nodeAddr := vars["nodeAddr"]

	ipaddr := net.ParseIP(nodeAddr).To4()
	n := datastore.ClusterNode{}
	err := n.NewClusterNode(ipaddr)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	return nil
}

func delClusterHandler(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	vars := mux.Vars(r)
	nodeAddr := vars[NODE_ADDRESS]
	ipaddr := net.ParseIP(nodeAddr).To4()
	n := datastore.ClusterNode{}
	err := n.RemoveClusterNode(ipaddr)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	return nil
}

func addBoostrapHandler(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	// TODO: Add boostrap / Listen config w/Interface arg
	vars := mux.Vars(r)
    // Interface to bind to
	nodeAddr := vars[BOOSTRAP_IFACE]
	ipaddr := net.ParseIP(nodeAddr).To4()
	n := datastore.ClusterNode{}
	err := n.NewClusterNode(ipaddr)
	if err != nil {
		return &apiError{http.StatusInternalServerError, err.Error()}
	}
	return nil
}
