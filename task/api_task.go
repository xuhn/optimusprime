package task

import (
	"../common"
	"../log"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var (
	apiTaskPoolMu sync.Mutex
	apiTaskPool   = make(map[int32]*APITask)
)

type APITask struct {
	Id         int32
	Gid        uint64
	Pattern    string
	Handler    ApiTaskHandler
	FuncName   string
	State      taskState
	timeOut    time.Duration
	isFinished chan bool
}

func NewAPITask(pattern string) (task *APITask, err error) {
	taskHandle, err := GetAPITaskHandle(pattern)
	if err != nil {
		return
	}
	task = &APITask{
		Id:         atomic.AddInt32(&globalTaskId, 1),
		Pattern:    pattern,
		Handler:    taskHandle.handler,
		State:      stateNew,
		timeOut:    taskHandle.timeOut,
		isFinished: make(chan bool),
	}
	apiTaskPoolMu.Lock()
	apiTaskPool[task.Id] = task
	apiTaskPoolMu.Unlock()
	return
}

func (t *APITask) Run(rw http.ResponseWriter, r *http.Request) (res []byte, err error) {
	t.setState(stateRun)

	funcName := GetTaskFuncName(t.Handler)
	go func() {
		t.Gid = common.GetGID()
		t.FuncName = funcName
		err := execute(rw, r, t.Handler)
		if err != nil {
			log.DEBUGF("[API_TASK(%d)|%s] execute err: %s", t.Id, funcName, err.Error())
		}
		t.isFinished <- true
	}()

	if t.timeOut > 0 {
		select {
		case <-t.isFinished:
			t.setState(stateFinished)
		case <-time.After(t.timeOut):
			t.setState(stateFinished)
			err = errors.New("task timet out")

		}
	} else {
		select {
		case <-t.isFinished:
			t.setState(stateFinished)
		}
	}

	apiTaskPoolMu.Lock()
	delete(apiTaskPool, t.Id)
	apiTaskPoolMu.Unlock()
	return
}

func (t *APITask) setState(state taskState) {
	t.State = state
}

func execute(rw http.ResponseWriter, r *http.Request, h ApiTaskHandler) (err error) {
	rw.Header().Set("Content-Type", "application/json; charset=utf-8")

	p := make(map[string]string)
	b := make([]byte, 0)

	if r.Header.Get("Content-Type") == "application/json" {
		b, err = ioutil.ReadAll(r.Body)
		if err != nil {
			return
		}

		r.Body.Close()
		p, err = parseJsonParams(b)
		if err != nil {
			return
		}
	} else {
		r.ParseForm()
		for key, values := range r.Form {
			p[key] = values[0]
		}
	}

	data, err := json.Marshal(h.ServeRequest(p))

	if err != nil {
		return
	}

	rw.Write(data)

	return
}

func parseJsonParams(b []byte) (ret map[string]string, err error) {
	ret = make(map[string]string)

	var f interface{}
	err = json.Unmarshal(b, &f)
	if err != nil {
		return
	}

	m := f.(map[string]interface{})
	for k, v := range m {
		switch vv := v.(type) {
		case string:
			ret[k] = vv
		case float64:
			ret[k] = strconv.FormatFloat(vv, 'f', -1, 64)
		}
	}
	return
}

func LenAPITasks() int {
	return len(apiTaskPool)
}

func GetAPITaskByGid(gid uint64) (task interface{}) {
	apiTaskPoolMu.Lock()
	defer apiTaskPoolMu.Unlock()
	for _, t := range apiTaskPool {
		if t.Gid == gid {
			return t
		}
	}
	return nil
}

func DumpAPITasks() (tasks map[int32]*APITask) {
	tasks = make(map[int32]*APITask)
	apiTaskPoolMu.Lock()
	defer apiTaskPoolMu.Unlock()
	for k, v := range apiTaskPool {
		tasks[k] = v
	}
	return
}
