package net

import (
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
)

// tcp
func ListenAndServeTCP(listen_addr string, listen_port int) (err error) {
	var listener net.Listener
	listen_ip, err := parseListenAddr(listen_addr)
	if err != nil {
		return
	}
	address := net.JoinHostPort(listen_ip, strconv.Itoa(listen_port))
	if listener, err = net.Listen("tcp", address); err != nil {
		return
	}
	server := newTcpServer(listener)
	server.serve()
	return
}

func SetTCPClientConnLimit(limit int) {
	setClientConnectionLimit(limit)
}

// 带回包的请求
func SendTCPRequest(s_peer_addr string, i_peer_port int, req []byte, timeOut uint32) (res []byte, err error) {
	return sendClientRequest(s_peer_addr, i_peer_port, req, timeOut)
}

// 不带回包的请求
func SendTCPRequestNoResponse(s_peer_addr string, i_peer_port int, req []byte, timeOut uint32) (err error) {
	return sendClientRequestNoResponse(s_peer_addr, i_peer_port, req, timeOut)
}

func SendTCPResponse(connection *TcpConnection, res []byte) (err error) {
	_, err = connection.Send(res)
	return
}

func ParseListenAddr(listen_addr string) (listen_ip string, err error) {
	return parseListenAddr(listen_addr)
}

func IsIPv4(ip string) bool {
	return isIPv4(ip)
}

func NewTcpConnection(conn net.Conn) *TcpConnection {
	return newTcpConnection(conn)
}

// http
func ListenAndServeHTTP(listen_addr string, listen_port int) (err error) {
	listen_ip, err := parseListenAddr(listen_addr)
	if err != nil {
		return
	}
	address := net.JoinHostPort(listen_ip, strconv.Itoa(listen_port))
	err = listenAndServeHTTP(address)
	return
}

// https
func ListenAndServeHTTPS(listen_addr string, listen_port int) (err error) {
	listen_ip, err := parseListenAddr(listen_addr)
	if err != nil {
		return
	}
	address := net.JoinHostPort(listen_ip, strconv.Itoa(listen_port))
	err = listenAndServeHTTPS(address)
	return
}

//自定义http multiplexer
func ListenAndServeHTTPMux(listen_addr string, listen_port int, mux http.Handler) (err error) {
	listen_ip, err := parseListenAddr(listen_addr)
	if err != nil {
		return
	}
	address := net.JoinHostPort(listen_ip, strconv.Itoa(listen_port))
	err = listenAndServeHTTPMux(address, mux)
	return
}

func SendHTTPRequest(uri string, params map[string]interface{}, timeOut uint32) (res []byte, err error) {
	return sendHttpRequest(uri, params, timeOut)
}

func SendHTTPPostRequest(uri string, body_type string, body io.Reader, timeOut uint32) (res []byte, err error) {
	return sendHttpPostRequest(uri, body_type, body, timeOut)
}

func SendHTTPMethodRequest(method, uri string, body io.Reader, timeOut uint32) (res []byte, err error) {
	return sendHttpMethodRequest(method, uri, body, timeOut)
}

//ratelimit
func InitRateLimit(strategy string, rate float32, capacity int64) error {
	if rate <= 0.0 || capacity <= 0 {
		return errors.New("ratelimiter input error")
	}
	defaultStrategy = strategy
	defaultCapacity = capacity
	defaultRate = rate
	return nil
}
