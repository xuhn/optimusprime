package net

import (
	"net/http"

	"github.com/xuhn/optimusprime/net/websocket"
)

var (
	RouteHTTP = func(w http.ResponseWriter, r *http.Request) {}
	RouteWs   = func(ws *websocket.Conn) {}
)
