package common

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
)

var (
	configContentByte []byte
	configContentJson map[string]interface{}
)

type block struct {
	data interface{}
}

func getConfig(data map[string]interface{}) (*block, error) {
	if data == nil {
		return nil, errors.New("get config fail, config content is nil")
	}
	return &block{
		data: data,
	}, nil
}

func (b *block) getValue(key string) *block {
	m := b.getData()
	if v, ok := m[key]; ok {
		b.data = v
		return b
	}
	return nil
}

func (b *block) getData() map[string]interface{} {
	if m, ok := (b.data).(map[string]interface{}); ok {
		return m
	}
	return nil
}

//读取配置
func LoadConfigFromFile(filename string) (err error) {
	configContentByte, err = ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	return json.Unmarshal(configContentByte, &configContentJson)
}

func LoadConfigFromData(data []byte) (err error) {
	configContentByte = data
	return json.Unmarshal(data, &configContentJson)
}

//获取配置value,支持按层次获取，点号分割
func GetConfigByKey(keys string) (value interface{}, err error) {
	key_list := strings.Split(keys, ".")
	block, err := getConfig(configContentJson)
	if err != nil {
		return nil, err
	}
	for _, key := range key_list {
		block = block.getValue(key)
		if block == nil {
			return nil, errors.New(fmt.Sprintf("can not get[\"%s\"]'s value", string(keys)))
		}
	}
	return block.data, nil
}

func StringDefault(key string, dfault string) string {
	return valueDefault(key, dfault).(string)
}

func IntDefault(key string, dfault int) int {
	miao := int(valueDefault(key, dfault).(float64))
	return miao
}

func BoolDefault(key string, dfault bool) bool {
	return valueDefault(key, dfault).(bool)
}

func valueDefault(key string, dfault interface{}) interface{} {
	_val, err := GetConfigByKey(key)
	if err != nil {
		return dfault
	}
	return _val
}

func DumpConfigContent() {
	var pjson bytes.Buffer
	json.Indent(&pjson, configContentByte, "", "\t")
	fmt.Println(string(pjson.Bytes()))
}
