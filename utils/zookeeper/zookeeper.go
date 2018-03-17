package zookeeper

import (
	"../log"
	"../utils/zookeeper/zk"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	ZkConnPoolMu sync.Mutex
	ZkConnPool   = make(map[string]*ZkConn)
)

type ZkConn struct {
	conn *zk.Conn
}

func initZkConn(connstr string) (zkConn *ZkConn, err error) {
	if connstr == "" {
		xflog.ERRORF("connect zk server failed:connection string is empty")
		err = errors.New("connection string is empty")
		return
	}
	conn_strs := strings.Split(connstr, ",")
	conn, ec, err := zk.Connect(conn_strs, 3*time.Second)
	if err != nil {
		xflog.ERRORF("connect zk server[\"%s\"] failed:%s", conn_strs, err)
		return
	}
	for {
		connEvent := <-ec
		switch connEvent.State {
		case zk.StateDisconnected:
			xflog.ERRORF("connect zk server:[\"%s\"] failed:%s", conn_strs, err)
			err = errors.New(fmt.Sprintf("connect zk server[\"%s\"] failed", conn_strs))
			return
		case zk.StateConnected:
			zkConn = &ZkConn{
				conn: conn,
			}
			ZkConnPoolMu.Lock()
			ZkConnPool[connstr] = zkConn
			ZkConnPoolMu.Unlock()
			return
		default:
			continue
		}
	}
}

func GetZkInstance(connstr string) (zkConn *ZkConn, err error) {
	ZkConnPoolMu.Lock()
	zkcon, ok := ZkConnPool[connstr]
	ZkConnPoolMu.Unlock()
	if ok {
		// 判断连接是否断开，如果断开，则重连
		if zkcon.conn.State() == zk.StateDisconnected {
			return initZkConn(connstr)
		}
		zkConn = zkcon
		return

	}
	return initZkConn(connstr)
}

func getConnstrByConn(conn *ZkConn) (connstr string, err error) {
	ZkConnPoolMu.Lock()
	defer ZkConnPoolMu.Unlock()
	for k, v := range ZkConnPool {
		if v == conn {
			connstr = k
			return
		}
	}
	err = errors.New("get connstr fail, conn is Invalid")
	return
}

func (c *ZkConn) CreateNode(path string, data []byte) (resPath string, err error) {
	if path == "" {
		return "", errors.New("Invalid Path")
	}
	// 判断连接是否断开
	if c.conn.State() == zk.StateDisconnected {
		connstr, err := getConnstrByConn(c)
		if err != nil {
			return "", err
		}
		c, err = GetZkInstance(connstr)
		if err != nil {
			return "", err
		}
		return c.CreateNode(path, data)
	}

	// 节点参数
	flag := int32(zk.FlagEphemeral)
	acl := zk.WorldACL(zk.PermAll)
	childPath := path

	// 创建父节点
	paths := strings.Split(path, "/")
	var parentPath string
	for _, v := range paths[1 : len(paths)-1] {
		parentPath += "/" + v
		exist, _, err := c.conn.Exists(parentPath)
		if err != nil {
			return "", err
		}
		if !exist {
			_, err = c.conn.Create(parentPath, nil, 0, acl) // 父节点必须是持久节点
			if err != nil {
				return "", err
			}
		}
	}

	// 创建子节点
	exist, _, err := c.conn.Exists(childPath)
	if err != nil {
		return "", err
	}
	if !exist {
		resPath, err = c.conn.Create(childPath, data, flag, acl)
		if err != nil {
			return "", err
		}
	} else {
		err = errors.New(fmt.Sprintf("[%s]  exists", childPath))
	}
	return
}

func (c *ZkConn) SetNode(path string, data []byte) (err error) {
	if path == "" {
		return errors.New("Invalid Path")
	}
	// 判断连接是否断开
	if c.conn.State() == zk.StateDisconnected {
		connstr, err := getConnstrByConn(c)
		if err != nil {
			return err
		}
		c, err = GetZkInstance(connstr)
		if err != nil {
			return err
		}
		return c.SetNode(path, data)
	}
	exist, stat, err := c.conn.Exists(path)
	if err != nil {
		return
	}
	if !exist {
		return errors.New(fmt.Sprintf("node [%s] dosen't exist,can't be setted", path))
	}
	_, err = c.conn.Set(path, data, stat.Version)
	if err != nil {
		return
	}
	return
}

func (c *ZkConn) GetNode(path string) (data []byte, err error) {
	// 判断路径是否为空
	if path == "" {
		return nil, errors.New("Invalid Path")
	}
	// 判断连接是否断开
	if c.conn.State() == zk.StateDisconnected {
		connstr, err := getConnstrByConn(c)
		if err != nil {
			return nil, err
		}
		c, err = GetZkInstance(connstr)
		if err != nil {
			return nil, err
		}
		return c.GetNode(path)
	}
	data, _, err = c.conn.Get(path)
	return
}

func (c *ZkConn) DeleteNode(path string) (err error) {
	// 判断路径是否为空
	if path == "" {
		return errors.New("Invalid Path")
	}
	// 判断连接是否断开
	if c.conn.State() == zk.StateDisconnected {
		connstr, err := getConnstrByConn(c)
		if err != nil {
			return err
		}
		c, err = GetZkInstance(connstr)
		if err != nil {
			return err
		}
		return c.DeleteNode(path)
	}
	// 判断节点是否存在
	exist, stat, err := c.conn.Exists(path)
	if err != nil {
		return err
	}
	if !exist {
		return errors.New(fmt.Sprintf("path [\"%s\"] doesn't exist", path))
	}
	// 删除节点
	return c.conn.Delete(path, stat.Version)
}

func (c *ZkConn) ListChildren(path string) (children []string, err error) {
	// 判断路径是否为空
	if path == "" {
		return nil, errors.New("Invalid Path")
	}
	// 判断连接是否断开
	if c.conn.State() == zk.StateDisconnected {
		connstr, err := getConnstrByConn(c)
		if err != nil {
			return nil, err
		}
		c, err = GetZkInstance(connstr)
		if err != nil {
			return nil, err
		}
		return c.ListChildren(path)
	}
	children, _, err = c.conn.Children(path)
	return
}

//获取当前节点（完整路径）的watcher
func (c *ZkConn) GetZNodeWatcher(path string) ([]byte, *zk.Stat, <-chan zk.Event, error) {
	return c.conn.GetW(path)
}

//获取当前节点所有子节点变化的wather
func (c *ZkConn) GetChildrenWatcher(path string) ([]string, *zk.Stat, <-chan zk.Event, error) {
	return c.conn.ChildrenW(path)
}
