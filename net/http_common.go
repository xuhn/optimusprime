package net

import (
	"optimusprime/net/websocket"
	"net/http"
)

var (
	RouteHTTP = func(w http.ResponseWriter, r *http.Request) {}
	RouteWs   = func(ws *websocket.Conn) {}
)
