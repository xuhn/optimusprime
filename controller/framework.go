package controller

import (
	"optimusprime/common"
	"optimusprime/log"
	"flag"
	"fmt"
)

var (
	confFile    = flag.String("c", "", "configuration file,json format")
	appName     = flag.String("a", "", "application name")
	confService = flag.String("s", "", "config service address,http server address")
)

var (
	BasePath string = "./"

	HTTPPort    int    // e.g. 9000
	HTTPAddr    string // e.g. "", "127.0.0.1"
	HTTPSsl     bool   // e.g. true if using ssl
	HTTPSslCert string // e.g. "/path/to/cert.pem"
	HTTPSslKey  string // e.g. "/path/to/key.pem"

	INFO  = log.INFO
	INFOF = log.INFOF
	WARN  = log.WARN
	WARNF = log.WARNF

	PrivKeyPath = "conf/app.rsa"
	PubKeyPath  = "conf/app.rsa.pub"
)

func Init() {
	// 解释命令行选项
	common.ProcessOptions()
	common.DumpOptions()
	// 处理配置
	option, err := common.GetOption("c")
	if err != nil {
		fmt.Println(err)
		option = "conf/config.json"
		//return
	}
	if err = common.LoadConfigFromFile(option); err != nil {
		fmt.Println("Load Config File fail,", err)
		return
	}
	common.DumpConfigContent()

	//初始化日志
	dir, _ := common.GetConfigByKey("log.dir")
	prefix, _ := common.GetConfigByKey("log.prefix")
	suffix, _ := common.GetConfigByKey("log.suffix")
	log_size, _ := common.GetConfigByKey("log.size")
	log_level, _ := common.GetConfigByKey("log.level")
	log.InitLogger(dir.(string), prefix.(string), suffix.(string), int64(log_size.(float64)), log_level.(string))

	HTTPPort = common.IntDefault("http.port", 9000)
	HTTPAddr = common.StringDefault("http.addr", "")
	HTTPSsl = common.BoolDefault("http.ssl", false)
	HTTPSslCert = common.StringDefault("http.sslcert", "")
	HTTPSslKey = common.StringDefault("http.sslkey", "")
	if HTTPSsl {
		if HTTPSslCert == "" {
			log.FATAL("No http.sslcert provided.")
		}
		if HTTPSslKey == "" {
			log.FATAL("No http.sslkey provided.")
		}
	}
}
