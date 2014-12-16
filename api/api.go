package api

import (
	"fmt"
	"log"
	"net/http"

	"github.com/docker/docker/vendor/src/github.com/gorilla/mux"
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
}

type Connection struct {
	ConnectionID  string `json:"connection_id"`
	ContainerID   string `json:"container_id"`
	ContainerName string `json:"container_name"`
	OvsPortID     string `json:"ovs_port_id"`
}

type HttpApiFunc func(w http.ResponseWriter, r *http.Request)

func Listen() {
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
			log.Printf("Registering %s, %s", method, route)
			r.Path(API_VERSION + route).Methods(method).HandlerFunc(fct)
		}
	}

	server := &http.Server{
		Addr:    ":6475",
		Handler: r,
	}
	server.ListenAndServe()
}

func getConfiguration(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func setConfiguration(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func getConnections(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func getConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	fmt.Println("Getting container id" + containerID)
	w.WriteHeader(http.StatusNotImplemented)
}

func createConnection(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func deleteConnection(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	containerID := vars["id"]
	fmt.Println("Deleting container id" + containerID)
	w.WriteHeader(http.StatusNotImplemented)
}
