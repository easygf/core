package config

import (
	"github.com/easygf/core/json"
	"github.com/easygf/core/log"
	"github.com/howeyc/fsnotify"
	"io"
	"os"
	"time"
)

const (
	ItemCreate = 1
	ItemUpdate = 2
	ItemDelete = 3
)

type Item struct {
	Key string
	Val string
	Ver int64
}

func tryGetLocalFs(key string) (*Item, error) {
	filePath := GetFilePathByKey(key)
	fp, err := os.Open(filePath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Errorf("err:%v", err)
		}
		return nil, err
	}
	defer func() {
		_ = fp.Close()
	}()
	buf, err := io.ReadAll(fp)
	if err != nil {
		log.Errorf("err:%v", err)
		return nil, err
	}
	if len(buf) == 0 {
		return nil, nil
	}
	var item Item
	err = json.Unmarshal(buf, &item)
	if err != nil {
		log.Errorf("err:%v", err)
		return nil, err
	}
	if item.Key == "" || item.Val == "" || item.Ver == 0 {
		log.Errorf("invalid item %+v", item)
		return nil, nil
	}
	return &item, nil
}

func (p *Item) ToJson(val interface{}) error {
	if p.Val != "" {
		err := json.Unmarshal([]byte(p.Val), val)
		if err != nil {
			log.Errorf("err:%v", err)
			return err
		}
	}
	return nil
}

type ItemWatcher struct {
	key        string
	callback   func(ev int, item *Item)
	notifyExit chan bool
	watcher    *fsnotify.Watcher
}

func NewItemWatcher(key string) *ItemWatcher {
	w := &ItemWatcher{key: key, watcher: nil}
	return w
}

func fileExist(filename string) bool {
	if _, err := os.Stat(filename); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

func (p *ItemWatcher) watchLoop() error {
	filePath := GetFilePathByKey(p.key)
	for {
		if p.watcher == nil {
			select {
			case <-p.notifyExit:
				return nil
			default:
			}
			if fileExist(filePath) {
				item, err := tryGetLocalFs(p.key)
				if err == nil && item != nil {
					if p.callback != nil {
						p.callback(ItemCreate, item)
					}
				}
				p.watcher, err = fsnotify.NewWatcher()
				if err != nil {
					log.Errorf("err:%v", err)
					time.Sleep(5 * time.Second)
					continue
				}
				err = p.watcher.Watch(filePath)
				if err != nil {
					log.Errorf("err:%v", err)
					_ = p.watcher.Close()
					p.watcher = nil
					time.Sleep(5 * time.Second)
					continue
				}
				log.Infof("watching %s", p.key)
			} else {
				time.Sleep(5 * time.Second)
				continue
			}
		} else {
			select {
			case ev := <-p.watcher.Event:
				if ev.IsDelete() {
					log.Warnf("key %s deleted", p.key)
					_ = p.watcher.Close()
					p.watcher = nil
					if p.callback != nil {
						p.callback(ItemDelete, nil)
					}
				} else if ev.IsAttrib() {
					// skip
				} else {
					log.Infof("%s update", p.key)
					newItem, err := tryGetLocalFs(p.key)
					if err == nil && newItem != nil {
						if p.callback != nil {
							p.callback(ItemUpdate, newItem)
						}
					}
				}
			case <-p.notifyExit:
				_ = p.watcher.Close()
				p.watcher = nil
				return nil
			}
		}
	}
}

func (p *ItemWatcher) Start(callback func(ev int, item *Item)) error {
	p.callback = callback
	if p.notifyExit == nil {
		p.notifyExit = make(chan bool, 1)
		routine.Go(nil, func(ctx *rpc.Context) error {
			err := p.watchLoop()
			if err != nil {
				log.Errorf("err:%v", err)
				return err
			}
			log.Infof("watch loop for key %s exit", p.key)
			return nil
		})
	}
	return nil
}

func (p *ItemWatcher) Stop() error {
	if p.notifyExit != nil {
		select {
		case p.notifyExit <- true:
		default:
		}
	}
	return nil
}
