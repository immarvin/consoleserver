package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/chenglch/consoleserver/common"
	"github.com/chenglch/consoleserver/console"
	"github.com/chenglch/consoleserver/storage"
	"github.com/gorilla/mux"
)

var (
	nodeManager  *console.NodeManager
	plog         = common.GetLogger("github.com/chenglch/consoleserver/api/node")
	serverConfig = common.GetServerConfig()
)

type NodeApi struct {
	routes Routes
}

func NewNodeApi(router *mux.Router) *NodeApi {
	api := NodeApi{}
	routes := Routes{
		Route{"Node", "GET", "/nodes", api.list},
		Route{"Node", "POST", "/nodes", api.post},
		Route{"Node", "GET", "/nodes/{node}", api.show},
		Route{"Node", "DELETE", "/nodes/{node}", api.delete},
		Route{"Node", "PUT", "/nodes/{node}", api.put},
		Route{"Node", "POST", "/bulk/nodes", api.bulkPost},
		Route{"Node", "DELETE", "/bulk/nodes", api.bulkDelete},
		Route{"Node", "PUT", "/bulk/nodes", api.bulkPut},
	}
	api.routes = routes
	for _, route := range routes {
		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(route.HandlerFunc)
	}
	nodeManager = console.GetNodeManager()
	return &api
}

func (api *NodeApi) list(w http.ResponseWriter, req *http.Request) {
	plog.Debug(fmt.Sprintf("Receive %s request %s from %s.", req.Method, req.URL.Path, req.RemoteAddr))
	var resp []byte
	var err error
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	nodes := nodeManager.ListNode()
	if resp, err = json.Marshal(nodes); err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	fmt.Fprintf(w, "%s\n", resp)
}

func (api *NodeApi) show(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	plog.Debug(fmt.Sprintf("Receive %s request %s %v from %s.", req.Method, req.URL.Path, vars, req.RemoteAddr))
	var resp []byte
	node, httpcode, err := nodeManager.ShowNode(vars["node"])
	if err != nil {
		plog.HandleHttp(w, req, httpcode, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	if resp, err = json.Marshal(node); err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s\n", resp)
}

func (api *NodeApi) put(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	plog.Debug(fmt.Sprintf("Receive %s request %s %v from %s.", req.Method, req.URL.Path, vars, req.RemoteAddr))
	var err error
	if !nodeManager.Exists(vars["node"]) {
		plog.HandleHttp(w, req, http.StatusBadRequest, err)
		return
	}
	if _, ok := req.URL.Query()["state"]; !ok {
		err = errors.New("Clould not locate the state parameters from URL")
		plog.HandleHttp(w, req, http.StatusBadRequest, err)
		return
	}
	state := req.URL.Query()["state"][0]
	nodes := make([]string, 0)
	nodes = append(nodes, vars["node"])
	nodeManager.SetConsoleState(nodes, state)
	plog.InfoNode(vars["node"], fmt.Sprintf("The console state has been changed to %s.", state))
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusAccepted)
}

func (api *NodeApi) bulkPut(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	plog.Debug(fmt.Sprintf("Receive %s request %s %v from %s.", req.Method, req.URL.Path, vars, req.RemoteAddr))
	var err error
	var resp []byte
	if _, ok := req.URL.Query()["state"]; !ok {
		err = errors.New("Clould not locate the state parameters from URL")
		plog.HandleHttp(w, req, http.StatusBadRequest, err)
		return
	}
	state := req.URL.Query()["state"][0]
	storNodes := make(map[string][]storage.Node, 0)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	if err := req.Body.Close(); err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	if err := json.Unmarshal(body, &storNodes); err != nil {
		plog.HandleHttp(w, req, http.StatusUnprocessableEntity, err)
		return
	}
	nodes := make([]string, len(storNodes["nodes"]))
	for _, v := range storNodes["nodes"] {
		nodes = append(nodes, v.Name)
	}
	result := nodeManager.SetConsoleState(nodes, state)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if resp, err = json.Marshal(result); err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "%s\n", resp)
}

func (api *NodeApi) post(w http.ResponseWriter, req *http.Request) {
	plog.Debug(fmt.Sprintf("Receive %s request %s from %s.", req.Method, req.URL.Path, req.RemoteAddr))
	node := storage.NewNode()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	if err := req.Body.Close(); err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	if err := json.Unmarshal(body, node); err != nil {
		plog.HandleHttp(w, req, http.StatusUnprocessableEntity, err)
		return
	}
	if node.Name == "" {
		plog.HandleHttp(w, req, http.StatusBadRequest, errors.New("Skip this record as node name is not defined"))
		return
	}
	if node.Driver == "" {
		plog.HandleHttp(w, req, http.StatusBadRequest, errors.New("Driver is not defined"))
		return
	}
	httpcode, err := nodeManager.PostNode(node)
	if err != nil {
		plog.HandleHttp(w, req, httpcode, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusCreated)
}

func (api *NodeApi) bulkPost(w http.ResponseWriter, req *http.Request) {
	plog.Debug(fmt.Sprintf("Receive %s request %s from %s.", req.Method, req.URL.Path, req.RemoteAddr))
	var resp []byte
	nodes := make(map[string][]storage.Node)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	if err := req.Body.Close(); err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	if err := json.Unmarshal(body, &nodes); err != nil {
		plog.HandleHttp(w, req, http.StatusUnprocessableEntity, err)
		return
	}
	result := nodeManager.PostNodes(nodes)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if resp, err = json.Marshal(result); err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, "%s\n", resp)
}

func (api *NodeApi) delete(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	plog.Debug(fmt.Sprintf("Receive %s request %s %v from %s.", req.Method, req.URL.Path, vars, req.RemoteAddr))
	httpcode, err := nodeManager.DeleteNode(vars["node"])
	if err != nil {
		plog.HandleHttp(w, req, httpcode, err)
		return
	}
	plog.InfoNode(vars["node"], "Deteled.")
	w.WriteHeader(http.StatusAccepted)
}

func (api *NodeApi) bulkDelete(w http.ResponseWriter, req *http.Request) {
	plog.Debug(fmt.Sprintf("Receive %s request %s from %s.", req.Method, req.URL.Path, req.RemoteAddr))
	var resp []byte
	nodes := make(map[string][]storage.Node, 0)
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	if err := req.Body.Close(); err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	if err := json.Unmarshal(body, &nodes); err != nil {
		plog.HandleHttp(w, req, http.StatusUnprocessableEntity, err)
		return
	}
	names := make([]string, len(nodes["nodes"]))
	for _, node := range nodes["nodes"] {
		names = append(names, node.Name)
	}
	result := nodeManager.DeleteNodes(names)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	if resp, err = json.Marshal(result); err != nil {
		plog.HandleHttp(w, req, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s\n", resp)
}
