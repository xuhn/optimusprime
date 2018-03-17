package main

import (
	"optimusprime/common"
	"optimusprime/controller"
	"optimusprime/log"
	"optimusprime/net"
	"fmt"
	_ "net/http/pprof"
	"os"
	_"reflect"
)

func listenAndServeTCP() {
	// 获取监听ip
	ip, err := common.GetConfigByKey("tcp.listen_addr")
	if err != nil {
		fmt.Println("can not get listen ip:", err)
		os.Exit(1)
	}
	listen_ip, _ := ip.(string)
	// 获取监听端口
	port, err := common.GetConfigByKey("tcp.listen_port")
	if err != nil {
		fmt.Println("can not get listen port")
		os.Exit(1)
	}
	listen_port := int(port.(float64))
	// 启动TCP服务
	if err = net.ListenAndServeTCP(listen_ip, listen_port); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	controller.Init()
	log.INFO("Running server...")
	/*
	controller.RegisterController((*api.User)(nil),
		[]*controller.MethodType{
			&controller.MethodType{
				Name: "Show",
				Args: []*controller.MethodArg{
					&controller.MethodArg{Name: "suite", Type: reflect.TypeOf((*string)(nil))},
				},
			},
		})

	controller.RegisterController((*api.Passport)(nil),
		[]*controller.MethodType{
			&controller.MethodType{
				Name: "Login",
				Args: []*controller.MethodArg{
				},
			},
		})
	*/
	controller.Run(0)
}
