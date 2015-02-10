package daemon

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/samalba/dockerclient"
)

type adapterRequest struct {
	PowerstripProtocolVersion int
	Type                      string
	ClientRequest             struct {
		Method  string
		Request string
		Body    string
	}
	ServerResponse struct {
		Body        string
		Code        int
		ContentType string
	}
}

type adapterPreResponse struct {
	PowerstripProtocolVersion int
	ModifiedClientRequest     struct {
		Method  string
		Request string
		Body    string
	}
}

type adapterPostResponse struct {
	PowerstripProtocolVersion int
	ModifiedServerResponse    struct {
		Body        string
		Code        int
		ContentType string
	}
}

func psAdapterPreHook(d *Daemon, reqParams adapterRequest) *adapterPreResponse {
	if reqParams.ClientRequest.Body != "" {
		jsonBody := &dockerclient.ContainerConfig{}
		err := json.Unmarshal([]byte(reqParams.ClientRequest.Body), &jsonBody)
		if err != nil {
			fmt.Println("Body JSON unmarshall failed", err)
		}

		jsonBody.HostConfig.NetworkMode = "none"

		preResp := &adapterPreResponse{}
		preResp.PowerstripProtocolVersion = reqParams.PowerstripProtocolVersion
		preResp.ModifiedClientRequest.Method = reqParams.ClientRequest.Method
		preResp.ModifiedClientRequest.Request = reqParams.ClientRequest.Request

		body, _ := json.Marshal(jsonBody)
		preResp.ModifiedClientRequest.Body = string(body)

		return preResp
	}
	return nil
}

func psAdapterPostHook(d *Daemon, reqParams adapterRequest) *adapterPostResponse {
	if reqParams.ClientRequest.Method != "POST" &&
		reqParams.ClientRequest.Method != "DELETE" {
		fmt.Println("Invalid method: ", reqParams.ClientRequest.Method)
		return nil
	}

	if reqParams.ClientRequest.Request != "" {
		// start api looks like this /<version>/containers/<cid>/start
		s := regexp.MustCompile("/").Split(reqParams.ClientRequest.Request, 5)
		cid := s[3]

		var cfg = &Connection{}
		var op = ConnectionAdd

		switch reqParams.ClientRequest.Method {
		case "POST":
			docker, _ := dockerclient.NewDockerClient(
				"unix:///var/run/docker.sock", nil)
			info, err := docker.InspectContainer(cid)
			if err != nil {
				fmt.Println("InspectContainer failed", err)
				return nil
			}

			cfg.ContainerID = string(cid)
			cfg.ContainerName = info.Name
			cfg.ContainerPID = strconv.Itoa(info.State.Pid)
			cfg.Network = DefaultNetworkName
			for _, env := range info.Config.Env {
				val := regexp.MustCompile("=").Split(env, 3)
				if val[0] == "SP_NETWORK" {
					cfg.Network = strings.Trim(val[1], " ")
				}
			}

			op = ConnectionAdd
		case "DELETE":

			var ok bool
			if cfg, ok = d.Connections[cid]; !ok {
				fmt.Println("Couldn't find the connection", cid)
				return nil
			}

			op = ConnectionDelete
		}

		context := &ConnectionContext{
			op,
			cfg,
			make(chan *Connection),
		}

		d.cC <- context

		<-context.Result

		postResp := &adapterPostResponse{}
		postResp.PowerstripProtocolVersion = reqParams.PowerstripProtocolVersion
		postResp.ModifiedServerResponse.ContentType = "application/json"
		postResp.ModifiedServerResponse.Body = reqParams.ServerResponse.Body
		postResp.ModifiedServerResponse.Code = reqParams.ServerResponse.Code

		return postResp
	}

	return nil
}

func psAdapter(d *Daemon, w http.ResponseWriter, r *http.Request) *apiError {
	var reqParams adapterRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&reqParams)
	if err != nil {
		fmt.Println("Error decodeing JSON", err)
		//return &apiError{http.StatusInternalServerError, err.Error()}
	}

	var data []byte
	switch reqParams.Type {
	case "pre-hook":
		data, _ = json.Marshal(psAdapterPreHook(d, reqParams))
	case "post-hook":
		data, _ = json.Marshal(psAdapterPostHook(d, reqParams))
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(data)
	return nil
}
