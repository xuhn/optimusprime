package task

import (
	"optimusprime/common"
	"optimusprime/log"
	"sync"
	"sync/atomic"
	"time"
)

var (
	timerTaskPoolMu sync.Mutex
	timerTaskPool   = make(map[int32]*TimerTask)
)

type TimerTask struct {
	Id           int32
	Gid          uint64
	Type         int32
	Handler      TimerTaskHandler
	FuncName     string
	State        taskState
	intervalTime time.Duration
	singleton    bool
	timer        *time.Ticker
	isFinished   chan bool
}

func newTimerTask(tType int32, handle *timerTaskHandle) (task *TimerTask, err error) {
	task = &TimerTask{
		Id:           atomic.AddInt32(&globalTaskId, 1),
		Type:         tType,
		Handler:      handle.handler,
		State:        stateNew,
		intervalTime: handle.intervalTime,
		singleton:    handle.singleton,
		isFinished:   make(chan bool),
	}
	timerTaskPoolMu.Lock()
	timerTaskPool[task.Id] = task
	timerTaskPoolMu.Unlock()
	return
}

func (t *TimerTask) Run() {
	funcName := GetTaskFuncName(t.Handler)
	go func() {
		t.Gid = common.GetGID()
		t.FuncName = funcName
		if common.CheckWrapPanic() {
			common.GoSafeTimer(t.Handler.ServeTimer)
		} else {
			t.Handler.ServeTimer()
		}
		t.isFinished <- true
	}()

	<-t.isFinished
	timerTaskPoolMu.Lock()
	delete(timerTaskPool, t.Id)
	timerTaskPoolMu.Unlock()
	return
}

func (t *TimerTask) setState(state taskState) {
	t.State = state
}

func TimerTaskServe() {
	timerTaskServeOnce.Do(timerTaskServe)
}

func timerTaskServe() {
	timerHandlePoolMu.Lock()
	defer timerHandlePoolMu.Unlock()
	for tType, handle := range timerHandlePool {
		// 先执行一次
		task, err := newTimerTask(tType, handle)
		if err != nil {
			log.ERRORF("create timer task[%d] fail", tType)
			continue
		}
		go task.Run()
		// 设置定时器
		taskTimer := time.NewTicker(handle.intervalTime)
		go runTimerTask(tType, handle, taskTimer)
	}
}

func runTimerTask(tType int32, handle *timerTaskHandle, t *time.Ticker) {
	for _ = range t.C {
		// 判断单例
		isRunning := false
		timerTaskPoolMu.Lock()
		for _, rtask := range timerTaskPool {
			if rtask.Type == tType {
				if rtask.singleton {
					isRunning = true
					break
				}
			}
		}
		timerTaskPoolMu.Unlock()
		if isRunning {
			continue
		}
		task, err := newTimerTask(tType, handle)
		if err != nil {
			log.ERRORF("create timer task fail, type(%d)", tType)
			continue
		}
		go task.Run()
	}
}

func LenTimerTasks() int {
	return len(timerTaskPool)
}

func GetTimerTaskByGid(gid uint64) (task interface{}) {
	timerTaskPoolMu.Lock()
	defer timerTaskPoolMu.Unlock()
	for _, t := range timerTaskPool {
		if t.Gid == gid {
			return t
		}
	}
	return nil
}

func DumpTimerTasks() (tasks map[int32]*TimerTask) {
	tasks = make(map[int32]*TimerTask)
	timerTaskPoolMu.Lock()
	defer timerTaskPoolMu.Unlock()
	for k, v := range timerTaskPool {
		tasks[k] = v
	}
	return
}
