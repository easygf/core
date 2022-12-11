package etcdutils

import (
	"context"
	"github.com/coreos/etcd/clientv3"
	"time"
)

type Kv struct {
	Key string
	Val string
	Ver int64
}

func SetWithVersion(
	cli *clientv3.Client, key, val string,
	version int64, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	txnRsp, err := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.Version(key), "=", version)).
		Then(clientv3.OpPut(key, val)).
		Commit()
	cancel()
	if err != nil {
		return false, err
	}
	return txnRsp.Succeeded, nil
}

func Set(cli *clientv3.Client, key, val string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	_, err := cli.Put(ctx, key, val)
	cancel()
	return err
}

func GetWithVersion(
	cli *clientv3.Client, key string,
	timeout time.Duration) (val string, version int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	rsp, err := cli.Get(ctx, key)
	cancel()
	val = ""
	version = 0
	if err == nil && len(rsp.Kvs) > 0 {
		val = string(rsp.Kvs[0].Value)
		version = rsp.Kvs[0].Version
	}
	return
}

func GetWithPrefix(cli *clientv3.Client, prefix string, timeout time.Duration) ([]*Kv, error) {
	var kvs []*Kv
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	rsp, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return nil, err
	}
	for _, n := range rsp.Kvs {
		kvs = append(
			kvs,
			&Kv{
				Key: string(n.Key),
				Val: string(n.Value),
				Ver: n.Version,
			})
	}
	return kvs, nil
}

func Del(cli *clientv3.Client, key string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	_, err := cli.Delete(ctx, key)
	cancel()
	return err
}
