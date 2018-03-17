package task

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type ApiTaskHandler interface {
	ServeRequest(params map[string]string) (result interface{})
}

type APITaskFunc func(params map[string]string) (result interface{})

func (t APITaskFunc) ServeRequest(params map[string]string) (result interface{}) {
	return t(params)
}

type apiTaskHandle struct {
	handler ApiTaskHandler
	timeOut time.Duration
}

var (
	apiHandlePoolMu sync.Mutex
	apiHandlePool   = make(map[string]*apiTaskHandle)
)

func RegisterAPITaskHandle(pattern string, handler ApiTaskHandler, timeOut time.Duration) {
	apiHandlePoolMu.Lock()
	defer apiHandlePoolMu.Unlock()
	newHandle := &apiTaskHandle{
		handler: handler,
		timeOut: timeOut,
	}
	apiHandlePool[pattern] = newHandle
}

func GetAPITaskHandle(pattern string) (*apiTaskHandle, error) {
	apiHandlePoolMu.Lock()
	defer apiHandlePoolMu.Unlock()
	if handle, ok := apiHandlePool[pattern]; ok {
		return handle, nil
	} else {
		return nil, errors.New("can't not find  handle")
	}
}

func DumpAPITaskHandle() {
	apiHandlePoolMu.Lock()
	defer apiHandlePoolMu.Unlock()
	for k, v := range apiHandlePool {
		fmt.Println(k, v)
	}
}
