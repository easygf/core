package config

import (
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/easygf/core/etcdclient"
	"github.com/easygf/core/etcdutils"
	"github.com/easygf/core/json"
	"github.com/easygf/core/log"
	"github.com/easygf/core/utils"
	"github.com/golang/protobuf/proto"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var Prefix = "config_"

func init() {
	err := os.MkdirAll(FsPath, 0777)
	if err != nil {
		fmt.Printf("make dir error, dir %s, error %s", FsPath, err)
		os.Exit(1)
	}
}

func GetFilePathByKey(key string) string {
	return filepath.Join(FsPath, key+".json")
}

type Config struct {
	cli *clientv3.Client
}

func NewConfig() *Config {
	return &Config{
		cli: nil,
	}
}

func (p *Config) EnsureConnected() (err error) {
	if p.cli == nil {
		p.cli, err = etcdclient.New()
		if err != nil {
			log.Error(err)
			return
		}
	}
	return
}

func (p *Config) Close() error {
	if p.cli != nil {
		err := p.cli.Close()
		p.cli = nil
		if err != nil {
			log.Errorf("err:%v", err)
		}
		return err
	}
	return nil
}

func (p *Config) CloseIgnoreError() {
	_ = p.Close()
}

func (p *Config) ListByPrefix(bizPrefix string, timeout time.Duration) ([]*Item, error) {
	err := p.EnsureConnected()
	if err != nil {
		log.Errorf("err:%v", err)
		return nil, err
	}
	prefix := Prefix + bizPrefix
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	list, err := etcdutils.GetWithPrefix(p.cli, prefix, timeout)
	if err != nil {
		log.Errorf("err:%v", err)
		return nil, err
	}
	var out []*Item
	for _, kv := range list {
		var key string
		if strings.HasPrefix(kv.Key, Prefix) {
			key = kv.Key[len(Prefix):]
		} else {
			key = kv.Key
		}
		out = append(out, &Item{
			Key: key,
			Val: kv.Val,
			Ver: kv.Ver,
		})
	}
	return out, nil
}

func (p *Config) Set(key string, val string) error {
	err := p.EnsureConnected()
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	err = etcdutils.Set(p.cli, Prefix+key, val, 3*time.Second)
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	log.Infof("set %s to %s", key, val)
	return nil
}

func (p *Config) SetJson(key string, val interface{}) error {
	var j []byte
	var err error
	if pb, ok := val.(proto.Message); ok {
		x, err := utils.Pb2JsonSkipDefaults(pb)
		if err != nil {
			log.Errorf("err:%v", err)
			return err
		}
		j = []byte(x)
	} else {
		j, err = json.Marshal(val)
		if err != nil {
			log.Errorf("err:%v", err)
			return err
		}
	}
	return p.Set(key, string(j))
}

func (p *Config) Del(key string) error {
	err := p.EnsureConnected()
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	err = etcdutils.Del(p.cli, Prefix+key, 3*time.Second)
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	return nil
}

func (p *Config) SetCheckVer(key string, val string, ver int64) error {
	err := p.EnsureConnected()
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	ok, err := etcdutils.SetWithVersion(p.cli, Prefix+key, val, ver, 3*time.Second)
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	if !ok {
		err = fmt.Errorf("key %s ver %d version out", key, ver)
		log.Error(err)
		return err
	}
	return nil
}

func (p *Config) Get(key string, fromLocalFs *bool) (*Item, error) {
	{
		item, err := tryGetLocalFs(key)
		if err == nil && item != nil {
			if fromLocalFs != nil {
				*fromLocalFs = true
			}
			return item, nil
		}
	}
	err := p.EnsureConnected()
	if err != nil {
		log.Errorf("err:%v", err)
		return nil, err
	}
	val, ver, err := etcdutils.GetWithVersion(
		p.cli, Prefix+key, 3*time.Second)
	if err != nil {
		c := etcdclient.GetEtcdConfig()
		log.Errorf("etcd get err %v, server %v", err, c.GetEndpointList())
		return nil, err
	}
	if fromLocalFs != nil {
		*fromLocalFs = false
	}
	return &Item{
		Key: key,
		Val: val,
		Ver: ver,
	}, nil
}

func (p *Config) GetJson(key string, val interface{}) (keyExisted bool, err error) {
	var i *Item
	i, err = p.Get(key, nil)
	if err != nil {
		log.Errorf("err:%v", err)
		return
	}
	if i.Val == "" {
		keyExisted = false
		return
	}
	err = json.Unmarshal([]byte(i.Val), val)
	if err != nil {
		log.Errorf("err:%v", err)
		return
	}
	return
}
