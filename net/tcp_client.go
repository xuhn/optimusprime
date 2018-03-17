package net

import (
	"optimusprime/log"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	maxBadConnRetries     = 2
	defaultConnectTimeout = 3 * time.Second
)

type clientTcpConnection struct {
	isOldConn bool
	tcpConn   *TcpConnection
}

var (
	clientTcpConnectionPoolMu sync.Mutex
	clientTcpConnectionPool       = make(map[string][]*clientTcpConnection)
	connectionLimit           int = 1000
)

func setClientConnectionLimit(limit int) {
	connectionLimit = limit
}

func newClientTcpConnection(s_peer_addr string, i_peer_port int, timeOut uint32) (c *clientTcpConnection, err error) {
	remote_addr := s_peer_addr + ":" + strconv.Itoa(i_peer_port)
	clientTcpConnectionPoolMu.Lock()
	connections, ok := clientTcpConnectionPool[remote_addr]
	if ok {
		connNum := len(connections)
		if connNum > 0 {
			c = connections[0]
			copy(connections, connections[1:])
			connections = connections[:connNum-1]
			clientTcpConnectionPool[remote_addr] = connections
			log.DEBUGF("get client connection [ %s -> %s ] from pool", c.tcpConn.conn.LocalAddr(), c.tcpConn.conn.RemoteAddr())
			clientTcpConnectionPoolMu.Unlock()
			return
		}
	}

	clientTcpConnectionPoolMu.Unlock()
	return connectServer("tcp", remote_addr, time.Duration(timeOut)*time.Second)
}

func connectServer(network, address string, timeout time.Duration) (c *clientTcpConnection, err error) {
	if timeout > defaultConnectTimeout {
		timeout = defaultConnectTimeout
	}
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return
	}
	log.DEBUGF("new client connection [ %s -> %s ]", conn.LocalAddr(), conn.RemoteAddr())
	tcpConn := newTcpConnection(conn)
	err = tcpConn.SetKeepAlive(defaultKeepAlivePeriod)
	c = &clientTcpConnection{
		tcpConn: tcpConn,
	}
	return
}

func freeClientTcpConnection(s_peer_addr string, i_peer_port int, c *clientTcpConnection) {
	remote_addr := s_peer_addr + ":" + strconv.Itoa(i_peer_port)
	clientTcpConnectionPoolMu.Lock()
	defer clientTcpConnectionPoolMu.Unlock()
	connections, ok := clientTcpConnectionPool[remote_addr]
	if !ok {
		connections = make([]*clientTcpConnection, 0)
	}
	connNum := len(connections)
	if connNum >= connectionLimit {
		c.tcpConn.Close()
	} else {
		c.isOldConn = true
		connections = append(connections, c)
		clientTcpConnectionPool[remote_addr] = connections
	}
}

func closeClientTcpConnection(s_peer_addr string, i_peer_port int) {
	remote_addr := s_peer_addr + ":" + strconv.Itoa(i_peer_port)
	clientTcpConnectionPoolMu.Lock()
	defer clientTcpConnectionPoolMu.Unlock()
	connections, ok := clientTcpConnectionPool[remote_addr]
	if ok {
		for _, v := range connections {
			v.tcpConn.Close()
		}
		delete(clientTcpConnectionPool, remote_addr)
	}
}

func sendClientRequest(s_peer_addr string, i_peer_port int, req []byte, timeOut uint32) ([]byte, error) {
	var connection *clientTcpConnection
	var res []byte
	var err error
	// 可能连接失效,重试
	for i := 0; i < maxBadConnRetries; i++ {
		connection, err = newClientTcpConnection(s_peer_addr, i_peer_port, timeOut)
		if err != nil {
			break
		}
		if err = connection.tcpConn.SetDeadline(time.Duration(timeOut) * time.Second); err != nil {
			if connection.isOldConn {
				closeClientTcpConnection(s_peer_addr, i_peer_port)
				continue
			}
			break
		}

		//发送
		writeErrChan := make(chan error)
		go func() {
			_, e := connection.tcpConn.Send(req)
			writeErrChan <- e
		}()

		select {
		case err = <-writeErrChan:
			//快速返回本地错误
			if err != nil {
				if connection.isOldConn {
					closeClientTcpConnection(s_peer_addr, i_peer_port)
					continue
				}
				break
			}
		case <-time.After(50 * time.Millisecond):
			err = <-writeErrChan
		}

		if err == nil {
			readErrChan := make(chan error)
			go func() {
				tmp_res, err := connection.tcpConn.Receive()
				res = make([]byte, 0, len(tmp_res))
				res = append(res, tmp_res...)
				//xflog.DEBUG("checking", len(res), res)
				readErrChan <- err
			}()

			select {
			case err = <-readErrChan:
				//快速返回本地错误
				if err != nil {
					if connection.isOldConn {
						closeClientTcpConnection(s_peer_addr, i_peer_port)
						continue
					}
					break
				}
			case <-time.After(50 * time.Millisecond):
				err = <-readErrChan
			}
			break
		}
		break
	}

	//放回连接池
	if err == nil && connection != nil {
		freeClientTcpConnection(s_peer_addr, i_peer_port, connection)
		return res, err
	}
	// 关闭连接
	if err != nil && connection != nil {
		connection.tcpConn.Close()
	}
	return res, err
}

func sendClientRequestNoResponse(s_peer_addr string, i_peer_port int, req []byte, timeOut uint32) (err error) {
	var connection *clientTcpConnection
	// 可能连接失效,重试
	for i := 0; i < maxBadConnRetries; i++ {
		connection, err = newClientTcpConnection(s_peer_addr, i_peer_port, timeOut)
		if err != nil {
			return
		}
		if err = connection.tcpConn.SetDeadline(time.Duration(timeOut) * time.Second); err != nil {
			if connection.isOldConn {
				closeClientTcpConnection(s_peer_addr, i_peer_port)
				continue
			}
			break
		}

		//发送
		writeErrChan := make(chan error)
		go func() {
			_, e := connection.tcpConn.Send(req)
			writeErrChan <- e
		}()

		select {
		case err = <-writeErrChan:
			//快速返回本地错误
			if err != nil {
				if connection.isOldConn {
					closeClientTcpConnection(s_peer_addr, i_peer_port)
					continue
				}
				break
			}
		case <-time.After(50 * time.Millisecond):
			err = <-writeErrChan
		}
		break
	}

	//放回连接池
	if err == nil && connection != nil {
		freeClientTcpConnection(s_peer_addr, i_peer_port, connection)
		return
	}
	// 关闭连接
	if err != nil && connection != nil {
		connection.tcpConn.Close()
	}
	return
}

func LenClientTcpConnections(s_peer_addr string, i_peer_port int) (plen int) {
	remote_addr := s_peer_addr + ":" + strconv.Itoa(i_peer_port)
	connections, ok := clientTcpConnectionPool[remote_addr]
	if ok {
		plen = len(connections)
	}
	return
}
