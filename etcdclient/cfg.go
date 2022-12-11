package etcdclient

import (
	"fmt"
	"github.com/easygf/core/json"
	"github.com/easygf/core/log"
	"os"
	"sync"
	"time"
)

type Node struct {
	Ip   string `json:"ip"`
	Port int    `json:"port"`
}
type EtcdConfig struct {
	Node             []Node `json:"node"`
	ConnectTimeoutMs int32  `json:"connect_timeout_ms"`
	OpTimeoutMs      int32  `json:"op_timeout_ms"`
	epList           []string
}

func (c *EtcdConfig) GetEndpointList() []string {
	return c.epList
}
func (c *EtcdConfig) GetOpTimeout() time.Duration {
	return time.Duration(c.OpTimeoutMs) * time.Millisecond
}
func (c *EtcdConfig) GetConnectTimeout() time.Duration {
	return time.Duration(c.ConnectTimeoutMs) * time.Millisecond
}

var ConfigPath string
var nodeList *EtcdConfig
var lastLoadTime int64
var mu sync.RWMutex

const loadIntervalSec = 30
const defaultConnectTimeoutMs = 800
const defaultOpTimeoutMs = 5000

func GetEtcdConfig() *EtcdConfig {
	now := time.Now().Unix()
	var nl *EtcdConfig
	mu.RLock()
	if nodeList != nil && lastLoadTime+loadIntervalSec >= now {
		nl = nodeList
	}
	mu.RUnlock()
	if nl != nil {
		return nl
	}
	mu.Lock()
	defer mu.Unlock()
	// double check, may be update by other goroutine
	if nodeList != nil && lastLoadTime+loadIntervalSec >= now {
		return nodeList
	}
	defer func() {
		lastLoadTime = now
	}()
	// read file
	dat, err := os.ReadFile(ConfigPath)
	if err != nil {
		log.Errorf("load etcd config file error, path %s, err %s", ConfigPath, err)
		goto OUT
	}
	nl = &EtcdConfig{}
	err = json.Unmarshal(dat, nl)
	if err != nil {
		log.Errorf("unmarsha1 etcd config file error, path %s, err %s", ConfigPath, err)
		goto OUT
	}
	if nl.ConnectTimeoutMs <= 0 {
		nl.ConnectTimeoutMs = defaultConnectTimeoutMs
	}
	if nl.OpTimeoutMs <= 0 {
		nl.OpTimeoutMs = defaultOpTimeoutMs
	}
	if len(nl.Node) == 0 {
		log.Errorf("node empty, path %s", ConfigPath)
		goto OUT
	}
	for _, n := range nl.Node {
		if n.Ip == "" || n.Port == 0 {
			log.Errorf("invalid node, node %+v", n)
			goto OUT
		}
		nl.epList = append(nl.epList, fmt.Sprintf("http://%s:%d", n.Ip, n.Port))
	}
	// swap
	nodeList = nl
OUT:
	if nodeList != nil {
		// return old one
		return nodeList
	}
	return &EtcdConfig{}
}
