package etcdclient

import (
	"errors"

	"github.com/coreos/etcd/clientv3"
)

func New() (*clientv3.Client, error) {
	c := GetEtcdConfig()
	epList := c.GetEndpointList()
	if len(epList) == 0 {
		return nil, errors.New("invalid etcd config")
	}
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   epList,
		DialTimeout: c.GetConnectTimeout(),
	})
	return cli, err
}