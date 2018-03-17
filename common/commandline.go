package common

import (
	"errors"
	"flag"
	"fmt"
)

func ProcessOptions() {
	if !flag.Parsed() {
		flag.Parse()
	}
}

func DumpOptions() {
	if !flag.Parsed() {
		flag.Parse()
	}
	visitor := func(a *flag.Flag) {
		fmt.Println("option=", a.Name, " value=", a.Value)
	}
	flag.Visit(visitor)
}

func GetOption(name string) (value string, err error) {
	flag := flag.Lookup(name)
	if flag == nil {
		return "", errors.New("can not find option")
	}
	if flag.Value.String() == "" {
		err = errors.New(fmt.Sprintf("option [\"%s\"] is empty", string(name)))
	}
	return flag.Value.String(), err
}
