package net

import (
	"net/http"

	"github.com/xuhn/optimusprime/net/websocket"
)

func listenAndServeHTTP(addr string) error {
	http.HandleFunc("/", RouteHTTP)
	http.Handle("/ws", websocket.Handler(RouteWs))
	return http.ListenAndServe(addr, nil)
}

func listenAndServeHTTPS(addr string) error {
	http.HandleFunc("/", RouteHTTP)
	//	http.Handle("/ws", websocket.Handler(RouteWs))
	return http.ListenAndServeTLS(addr, "server.crt", "server.key", nil)
}

/*
	自定义multiplexer，使用方式
	mux中可包括:
	mux.HandleFunc("/", RouteHTTP)
	mux.Handle("/ws", websocket.Handler(RouteWs))
*/
func listenAndServeHTTPMux(addr string, mux http.Handler) error {
	return http.ListenAndServe(addr, mux)
}
