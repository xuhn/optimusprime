package task

import (
	"optimusprime/common"
	"optimusprime/log"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

type taskState int

const (
	stateNew taskState = iota
	stateRun
	stateFinished
)

var (
	globalTaskId       int32
	taskStatServeOnce  *sync.Once    = &sync.Once{}
	timerTaskServeOnce *sync.Once    = &sync.Once{}
	statIntervalTime   time.Duration = 10 * time.Second
)

// 根据handler获取任务方法名字
func GetTaskFuncName(taskHandler interface{}) string {
	funcInfo := runtime.FuncForPC(reflect.ValueOf(taskHandler).Pointer()).Name()
	return strings.Split(funcInfo, ".")[1]
}

// 根据Goroutine Id 获取任务实例
func GetTaskByGid(gid uint64) (task interface{}) {
	task = GetTCPTaskByGid(gid)
	if task != nil {
		return
	}
	task = GetTimerTaskByGid(gid)
	if task != nil {
		return
	}
	task = GetHTTPTaskByGid(gid)
	return
}

// ===================================================================================
// 任务日志
func T_DEBUGF(format string, v ...interface{}) {
	taskLog("DEBUG", format, v...)
}

func T_NFOF(format string, v ...interface{}) {
	taskLog("INFO", format, v...)
}

func T_WARNF(format string, v ...interface{}) {
	taskLog("WARN", format, v...)
}

func T_ERRORF(format string, v ...interface{}) {
	taskLog("ERROR", format, v...)
}

func taskLog(level string, format string, v ...interface{}) {
	var newFormat string
	// 根据goroutine id获取task 信息
	gid := common.GetGID()
	task := GetTaskByGid(gid)
	// 解析对应的task信息并写入日志,格式[xxx(xxx)|func(xxx)]
	switch task.(type) {
	case *TCPTask:
		tcpTask := task.(*TCPTask)
		funcName := tcpTask.FuncName
		newFormat = fmt.Sprintf("[TCP_TASK(%d)|%s] %s", tcpTask.Id, funcName, format)
	case *TimerTask:
		timerTask := task.(*TimerTask)
		funcName := timerTask.FuncName
		newFormat = fmt.Sprintf("[TIMER_TASK(%d)|%s] %s", timerTask.Id, funcName, format)
	case *HTTPTask:
		httpTask := task.(*HTTPTask)
		funcName := httpTask.FuncName
		newFormat = fmt.Sprintf("[HTTP_TASK(%d)|%s] %s", httpTask.Id, funcName, format)
	}

	switch level {
	case "DEBUG":
		log.DEBUGF(newFormat, v...)
	case "INFO":
		log.INFOF(newFormat, v...)
	case "WARN":
		log.WARNF(newFormat, v...)
	case "ERROR":
		log.ERRORF(newFormat, v...)
	}
}

// ===================================================================================

// 定时统计运行任务信息
func TaskStatServe() {
	go taskStatServe(taskStatServeOnce)
}

func taskStatServe(once *sync.Once) {
	once.Do(func() {
		statTimer := time.NewTicker(statIntervalTime)
		for _ = range statTimer.C {
			printRunningTaskStat()
		}
	})
}

func printRunningTaskStat() {
	table := common.NewTable([]string{"Type", "Id", "FuncName"})
	// 统计TCP 任务
	tcpTasks := DumpTCPTasks()
	for k, v := range tcpTasks {
		row := map[string]interface{}{
			"Type":     "TCP_TASK",
			"Id":       k,
			"FuncName": GetTaskFuncName(v.Handler),
		}
		table.AddRow(row)
	}

	// 统计定时任务
	timerTasks := DumpTimerTasks()
	for k, v := range timerTasks {
		row := map[string]interface{}{
			"Type":     "TIMER_TASK",
			"Id":       k,
			"FuncName": GetTaskFuncName(v.Handler),
		}
		table.AddRow(row)
	}
	// 统计HTTP任务
	httpTasks := DumpHTTPTasks()
	for k, v := range httpTasks {
		row := map[string]interface{}{
			"Type":     "HTTP_TASK",
			"Id":       k,
			"FuncName": GetTaskFuncName(v.Handler),
		}
		table.AddRow(row)
	}
	table.Print()
}
