package common

import (
	"optimusprime/log"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
)

func AESDecrypt(data, key, iv string) ([]byte, error) {
	//被加密的数据
	decodeBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, err
	}
	//加密秘钥
	keyByte, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	//偏移量
	ivByte, err := base64.StdEncoding.DecodeString(iv)
	if err != nil {
		return nil, err
	}
	// 创建加密算法apes
	block, err := aes.NewCipher(keyByte)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, ivByte)
	// CryptBlocks可以原地更新
	mode.CryptBlocks(decodeBytes, decodeBytes)
	return decodeBytes, nil
}

func AESDecrypt2Obj(data, key, iv string, result interface{}) error {
	_result, err := AESDecrypt(data, key, iv)
	if err != nil {
		return err
	}
	log.DEBUGF("AESDecrypt: %v", string(_result))
	err = json.NewDecoder(bytes.NewReader(_result)).Decode(result)
	if err != nil {
		return err
	}
	return nil
}
