package storage

import (
	"errors"
	"fmt"
	"github.com/chenglch/consoleserver/common"
)

var (
	serverConfig     = common.GetServerConfig()
	plog             = common.GetLogger("github.com/chenglch/consoleserver/storage")
	STORAGE_INIT_MAP = map[string]func() StorInterface{}
)

type Node struct {
	Name     string            `json:"name"`
	Driver   string            `json:"driver"` // node type cmd, ssh, ipmitool
	Params   map[string]string `json:"params"`
	Ondemand bool              `json:"ondemand, true"`
}

func NewNode() *Node {
	return new(Node)
}

type Storage struct {
	Nodes map[string]*Node
	async bool
}

func (s *Storage) GetNodes() map[string]*Node {
	return s.Nodes
}

type StorInterface interface {
	ImportNodes()
	PersistWatcher(eventChan chan map[int][]byte)
	GetNodes() map[string]*Node
	NotifyPersist(interface{}, int)
	IsAsync() bool
	ListNodeWithHost() map[string]string
}

func NewStorage(storType string) (StorInterface, error) {
	if _, ok := STORAGE_INIT_MAP[storType]; !ok {
		return nil, errors.New(fmt.Sprintf("The storage type %s is not exist", storType))
	}
	return STORAGE_INIT_MAP[storType](), nil
}
