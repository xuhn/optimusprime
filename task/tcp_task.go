package task

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/xuhn/optimusprime/common"
)

var (
	tcpTaskPoolMu sync.Mutex
	tcpTaskPool   = make(map[int32]*TCPTask)
)

type TCPTask struct {
	Id       int32
	Gid      uint64
	Type     int32
	Handler  TCPTaskHandler
	FuncName string
	State    taskState
	timeOut  time.Duration
	msgChan  chan []byte
}

func NewTCPTask(tType int32) (task *TCPTask, err error) {
	taskHandle, err := GetTCPTaskHandle(tType)
	if err != nil {
		return
	}
	task = &TCPTask{
		Id:      atomic.AddInt32(&globalTaskId, 1),
		Type:    tType,
		Handler: taskHandle.handler,
		State:   stateNew,
		timeOut: taskHandle.timeOut,
		msgChan: make(chan []byte),
	}
	tcpTaskPoolMu.Lock()
	tcpTaskPool[task.Id] = task
	tcpTaskPoolMu.Unlock()
	return
}

func (t *TCPTask) Run(req interface{}) (res []byte, err error) {
	t.setState(stateRun)
	funcName := GetTaskFuncName(t.Handler)
	var ok bool
	go func() {
		t.Gid = common.GetGID()
		t.FuncName = funcName
		if common.CheckWrapPanic() {
			common.GoSafeTCP(t.msgChan, req, t.Handler.ServeTCP)
		} else {
			t.Handler.ServeTCP(t.msgChan, req)
		}
	}()

	if t.timeOut > 0 {
		select {
		case res, ok = <-t.msgChan:
			if !ok {
				err = errors.New("task fail ,close")
			}
			t.setState(stateFinished)
		case <-time.After(t.timeOut):
			t.setState(stateFinished)
			err = errors.New("task timet out")

		}
	} else {
		select {
		case res, ok = <-t.msgChan:
			if !ok {
				err = errors.New("task fail ,close")
			}
			t.setState(stateFinished)
		}
	}
	tcpTaskPoolMu.Lock()
	delete(tcpTaskPool, t.Id)
	tcpTaskPoolMu.Unlock()
	return
}

func (t *TCPTask) setState(state taskState) {
	t.State = state
}

func LenTCPTasks() int {
	return len(tcpTaskPool)
}

func GetTCPTaskByGid(gid uint64) (task interface{}) {
	tcpTaskPoolMu.Lock()
	defer tcpTaskPoolMu.Unlock()
	for _, t := range tcpTaskPool {
		if t.Gid == gid {
			return t
		}
	}
	return nil
}

func DumpTCPTasks() (tasks map[int32]*TCPTask) {
	tasks = make(map[int32]*TCPTask)
	tcpTaskPoolMu.Lock()
	defer tcpTaskPoolMu.Unlock()
	for k, v := range tcpTaskPool {
		tasks[k] = v
	}
	return
}
