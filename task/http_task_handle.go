package task

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type httpTaskHandle struct {
	handler http.Handler
	timeOut time.Duration
}

var (
	httpHandlePoolMu sync.Mutex
	httpHandlePool   = make(map[string]*httpTaskHandle)
)

func RegisterHTTPTaskHandle(pattern string, handler http.Handler, timeOut time.Duration) {
	httpHandlePoolMu.Lock()
	defer httpHandlePoolMu.Unlock()
	newHandle := &httpTaskHandle{
		handler: handler,
		timeOut: timeOut,
	}
	httpHandlePool[pattern] = newHandle
}

func GetHTTPTaskHandle(pattern string) (*httpTaskHandle, error) {
	httpHandlePoolMu.Lock()
	defer httpHandlePoolMu.Unlock()
	if handle, ok := httpHandlePool[pattern]; ok {
		return handle, nil
	} else {
		return nil, errors.New("can't not find  handle")
	}
}

func DumpHTTPTaskHandle() {
	httpHandlePoolMu.Lock()
	defer httpHandlePoolMu.Unlock()
	for k, v := range httpHandlePool {
		fmt.Println(k, v)
	}
}
