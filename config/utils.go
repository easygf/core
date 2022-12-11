package config

import (
	"context"
	"errors"
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/easygf/core/etcdclient"
	"github.com/easygf/core/etcdutils"
	"github.com/easygf/core/json"
	"github.com/easygf/core/log"
)

func GetStr(key string) (val string, ver int64, err error) {
	var cli *clientv3.Client
	cli, err = etcdclient.New()
	if err != nil {
		log.Error(err)
		return
	}
	defer func(cli *clientv3.Client) {
		err := cli.Close()
		if err != nil {
			log.Error(err)
		}
	}(cli)
	var realKey string
	realKey = Prefix + key
	val, ver, err = etcdutils.GetWithVersion(
		cli, realKey, etcdclient.GetEtcdConfig().GetOpTimeout())
	if err != nil {
		log.Error(err)
		return
	}
	return
}

func GetStrPrefix(key, prefix string) (val string, ver int64, err error) {
	var cli *clientv3.Client
	cli, err = etcdclient.New()
	if err != nil {
		log.Error(err)
		return
	}
	defer func(cli *clientv3.Client) {
		err := cli.Close()
		if err != nil {
			log.Error(err)
		}
	}(cli)
	var realKey string
	realKey = prefix + key
	val, ver, err = etcdutils.GetWithVersion(
		cli, realKey, etcdclient.GetEtcdConfig().GetOpTimeout())
	if err != nil {
		log.Error(err)
		return
	}
	return
}

func SetStr(key string, val string, ver int64) (err error) {
	var cli *clientv3.Client
	cli, err = etcdclient.New()
	if err != nil {
		log.Error(err)
		return
	}
	defer func(cli *clientv3.Client) {
		err := cli.Close()
		if err != nil {
			log.Error(err)
		}
	}(cli)
	realKey := Prefix + key
	var res bool
	res, err = etcdutils.SetWithVersion(
		cli, realKey, val, ver, etcdclient.GetEtcdConfig().GetOpTimeout())
	if err != nil {
		log.Error(err)
		return
	}
	if !res {
		err = errors.New("version out")
		log.Error(err)
		return
	}
	return
}

func SetStrPrefix(key, prefix, val string, ver int64) (err error) {
	var cli *clientv3.Client
	cli, err = etcdclient.New()
	if err != nil {
		log.Error(err)
		return
	}
	defer func(cli *clientv3.Client) {
		err := cli.Close()
		if err != nil {
			log.Error(err)
		}
	}(cli)
	realKey := prefix + key
	var res bool
	res, err = etcdutils.SetWithVersion(
		cli, realKey, val, ver, etcdclient.GetEtcdConfig().GetOpTimeout())
	if err != nil {
		log.Error(err)
		return
	}
	if !res {
		err = errors.New("version out")
		log.Error(err)
		return
	}
	return
}

func Del(key string) (err error) {
	var cli *clientv3.Client
	cli, err = etcdclient.New()
	if err != nil {
		log.Error(err)
		return
	}
	defer func(cli *clientv3.Client) {
		err := cli.Close()
		if err != nil {
			log.Error(err)
		}
	}(cli)
	realKey := Prefix + key
	err = etcdutils.Del(
		cli, realKey, etcdclient.GetEtcdConfig().GetOpTimeout())
	if err != nil {
		log.Error(err)
		return
	}
	return
}

func Watch(key string, cb func(val string, isDelete bool) (stopWatch bool)) error {
	cli, err := etcdclient.New()
	if err != nil {
		return err
	}
	defer func(cli *clientv3.Client) {
		err := cli.Close()
		if err != nil {
			log.Error(err)
		}
	}(cli)
	realKey := Prefix + key
	watchChan := cli.Watch(context.Background(), realKey)
	for rsp := range watchChan {
		for _, ev := range rsp.Events {
			if ev.Type == mvccpb.PUT {
				stop := cb(string(ev.Kv.Value), false)
				if stop {
					return nil
				}
			} else if ev.Type == mvccpb.DELETE {
				stop := cb("", true)
				if stop {
					return nil
				}
			}
		}
	}
	return nil
}

func GetJsonField(key string, fieldName string) (res interface{}, err error) {
	var val string
	val, _, err = GetStr(key)
	if err != nil {
		log.Error(err)
		return
	}
	m := make(map[string]interface{})
	err = json.Unmarshal([]byte(val), &m)
	if err != nil {
		log.Errorf("json decode fail, `%s` %s", val, err)
		return
	}
	if v, ok := m[fieldName]; ok {
		res = v
	}
	return
}
