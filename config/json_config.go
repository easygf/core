package config

import (
	"errors"
	"fmt"
	"github.com/easygf/core/log"
	"reflect"
	"sync"
)

type JsonConfig struct {
	key     string
	typ     reflect.Type
	val     interface{}
	existed bool
	hasInit bool
}

type ChangedCb func(item int, oldVal interface{}, newVal interface{})

func NewJsonConfig(key string, typeInstance interface{}) *JsonConfig {
	t := reflect.TypeOf(typeInstance)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if key == "" {
		panic("invalid key")
	}
	return &JsonConfig{
		key:     key,
		typ:     t,
		val:     nil,
		hasInit: false,
	}
}

func (p *JsonConfig) Init() error {
	return p.InitV2(nil)
}

func (p *JsonConfig) InitV2(logic ChangedCb) error {
	if p.hasInit {
		return nil
	}
	defer func() {
		p.hasInit = true
	}()
	p.val = reflect.New(p.typ).Interface()
	if rpc.Meta.IsDevRole() && !rpc.Meta.UseEtcdInDev {
		return nil
	}
	c := NewConfig()
	defer c.CloseIgnoreError()
	existed, err := c.GetJson(p.key, p.val)
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	p.existed = existed
	watcher := NewItemWatcher(p.key)
	err = watcher.Start(func(ev int, item *Item) {
		old := p.val
		v := reflect.New(p.typ).Interface()
		switch ev {
		case ItemCreate, ItemUpdate:
			p.existed = true
			err = item.ToJson(v)
			if err != nil {
				log.Errorf("err:%v", err)
			} else {
				p.val = v
			}
		case ItemDelete:
			p.existed = false
		}
		if logic != nil {
			logic(ev, old, p.val)
		}
	})
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	return nil
}

func (p *JsonConfig) Existed() bool {
	return p.existed
}

func (p *JsonConfig) Get() interface{} {
	return p.val
}

func (p *JsonConfig) Set(val interface{}) error {
	c := NewConfig()
	defer func(c *Config) {
		err := c.Close()
		if err != nil {
			// do nothing
		}
	}(c)
	err := c.SetJson(p.key, val)
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	p.val = val
	return nil
}

type JsonConfigList struct {
	val          map[uint32]*JsonConfig
	prefix       string
	typeInstance interface{}
	lock         sync.RWMutex
}

var NotFoundErr = errors.New("not found typ")

func NewJsonConfigList(prefix string, typeInstance interface{}) *JsonConfigList {
	return &JsonConfigList{
		val:          map[uint32]*JsonConfig{},
		prefix:       prefix,
		typeInstance: typeInstance,
	}
}

func (j *JsonConfigList) AddNew(typ uint32) error {
	j.lock.Lock()
	defer j.lock.Unlock()
	if _, ok := j.val[typ]; ok {
		return nil
	}
	cfg := NewJsonConfig(j.genKey(typ), j.typeInstance)
	err := cfg.Init()
	if err != nil {
		log.Errorf("err:%v", err)
		return err
	}
	j.val[typ] = cfg
	return nil
}
func (j *JsonConfigList) genKey(typ uint32) string {
	return fmt.Sprintf("%s_%d", j.prefix, typ)
}
func (j *JsonConfigList) Get(typ uint32) (interface{}, error) {
	if v, ok := j.val[typ]; ok {
		return v, nil
	} else {
		return nil, NotFoundErr
	}
}
func (j *JsonConfigList) Set(typ uint32, val interface{}) error {
	if v, ok := j.val[typ]; ok {
		return v.Set(val)
	} else {
		return NotFoundErr
	}
}
func (j *JsonConfigList) Existed(typ uint32) bool {
	if v, ok := j.val[typ]; ok {
		return v.Existed()
	} else {
		return false
	}
}
func (j *JsonConfigList) GetAll2Map() map[uint32]interface{} {
	j.lock.RLock()
	defer j.lock.RUnlock()
	m := map[uint32]interface{}{}
	for k, v := range j.val {
		m[k] = v.Get()
	}
	return m
}
