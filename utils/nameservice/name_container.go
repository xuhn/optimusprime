package nameservice

import (
	"../log"
	"../message/proto/uns"
	"../message/protobuf/proto"
	"../utils/zookeeper"
	"errors"
	"math/rand"
	"sync"
)

type NameContainer struct {
	sync.RWMutex
	MapNNC        map[string]*uns.NameNodeContent
	MapNameValues map[string][]string
}

func NewNameContainer() *NameContainer {
	nc := new(NameContainer)
	nc.MapNNC = make(map[string]*uns.NameNodeContent)
	nc.MapNameValues = make(map[string][]string)
	return nc
}

func (nc *NameContainer) SetName(shortName, fullName string, nnc *uns.NameNodeContent) {
	nc.Lock()
	defer nc.Unlock()
	nc.MapNNC[fullName] = nnc
	nc.MapNameValues[shortName] = append(nc.MapNameValues[shortName], fullName)
}

func (nc *NameContainer) GetName(shortName string) (interface{}, interface{}, error) {
	nc.RLock()
	defer nc.RUnlock()

	if _, ok := nc.MapNameValues[shortName]; !ok {
		return nil, nil, errors.New("shortname to zk node empty")
	}
	if len(nc.MapNameValues[shortName]) == 0 {
		return nil, nil, errors.New("shortname length to zk node empty")
	}
	num := rand.Intn(len(nc.MapNameValues[shortName]))
	full_name := nc.MapNameValues[shortName][num]
	if full_name == "" {
		xflog.ERRORF("[name_container]no full name for short name %s", shortName)
		return nil, nil, errors.New("no full name")
	}
	nnc, ok := nc.MapNNC[full_name]
	if !ok {
		xflog.ERRORF("[name_container]no name node content for full name %s", full_name)
		return nil, nil, errors.New("no name node")
	}
	ip := nnc.GetIp()
	port := int(nnc.GetPort())
	return ip, port, nil
}

func (nc *NameContainer) GetNameBatch(shortName string) ([]uns.NameNodeContent, error) {
	nc.RLock()
	defer nc.RUnlock()
	if _, ok := nc.MapNameValues[shortName]; !ok {
		return nil, errors.New("shortname to zk node empty")
	}
	if len(nc.MapNameValues[shortName]) == 0 {
		return nil, errors.New("shortname length to zk node empty")
	}
	uns_lst := make([]uns.NameNodeContent, 0, len(nc.MapNameValues[shortName]))
	for _, full_name := range nc.MapNameValues[shortName] {
		node := nc.MapNNC[full_name]
		uns_node := uns.NameNodeContent{
			Ip:       proto.String(node.GetIp()),
			Port:     proto.Uint32(node.GetPort()),
			Reserved: node.GetReserved(),
		}
		uns_lst = append(uns_lst, uns_node)
	}
	return uns_lst, nil
}

func (nc *NameContainer) ClearNameValues(shortName string) {
	nc.Lock()
	defer nc.Unlock()
	nc.MapNameValues[shortName] = make([]string, 0)
}

func (nc *NameContainer) SetNameBatch(shortName string, fullNames []string, nameNodes []*uns.NameNodeContent) {
	nc.Lock()
	defer nc.Unlock()
	nc.MapNameValues[shortName] = make([]string, 0)
	if len(fullNames) == 0 || len(nameNodes) == 0 {
		return
	}
	for k, _ := range fullNames {
		nc.MapNNC[fullNames[k]] = nameNodes[k]
		nc.MapNameValues[shortName] = append(nc.MapNameValues[shortName], fullNames[k])
	}
}

func (nc *NameContainer) FetchZkName(connStr, shortName, fullName string) error {
	zk, err := zookeeper.GetZkInstance(connStr)
	if err != nil {
		xflog.ERROR("get zk instance error")
		return err
	}
	childs, _, ch, err := zk.GetChildrenWatcher(fullName)
	if err != nil {
		xflog.ERRORF("get children %s instance error", fullName)
		return err
	}
	fullNames := make([]string, 0)
	nameNodes := make([]*uns.NameNodeContent, 0)
	for _, v := range childs {
		full_node := fullName + "/" + v
		data, err := zk.GetNode(full_node)
		if err != nil {
			xflog.ERROR("get child data err", err)
			continue
		}
		msg := &uns.NameNodeContent{}
		err = proto.Unmarshal(data, msg)
		if err != nil {
			xflog.ERROR("parse name node content error", full_node, err)
			continue
		}
		fullNames = append(fullNames, full_node)
		nameNodes = append(nameNodes, msg)
	}
	nc.SetNameBatch(shortName, fullNames, nameNodes)
	go func() {
		select {
		case ev := <-ch:
			xflog.INFO("cat watcher", ev)
			nc.FetchZkName(connStr, shortName, fullName)
		}
	}()
	return nil
}

var (
	nameContainer = NewNameContainer()
)

func InitNameService(zk_server string, name_lst map[string]interface{}) {
	updateNames(zk_server, name_lst)
	return
}

func updateNames(zk_server string, name_lst map[string]interface{}) {
	for k, v := range name_lst {
		short_name := k
		full_name := v.(string)
		err := nameContainer.FetchZkName(zk_server, short_name, full_name)
		if err != nil {
			xflog.ERROR("[update_names]fetch node name error: ", err)
			continue
		}
	}
	xflog.DEBUG("[update_names]name container info:", nameContainer)
	return
}

func GetInstance(shortname string) (interface{}, interface{}) {
	ip, port, err := nameContainer.GetName(shortname)
	if err != nil {
		xflog.ERROR(err)
		return nil, nil
	}
	return ip, port
}

func GetAllInstance(shortname string) ([]uns.NameNodeContent, error) {
	return nameContainer.GetNameBatch(shortname)
}

//zk注册临时节点，session保持的情况下不会删除节点，当session断开后节点不存在
func RegisterOnce(zk_server, zk_node, ip string, port uint32) {
	registerNode(zk_server, zk_node, ip, port)
	return
}

func registerNode(zk_server, zk_node, ip string, port uint32) {
	zk, err := zookeeper.GetZkInstance(zk_server)
	if err != nil {
		xflog.ERROR("[register_myself]connect zk server error")
		return
	}
	reg_node := &uns.NameNodeContent{
		Ip:   proto.String(ip),
		Port: proto.Uint32(port),
	}
	bin_reg_node, err := proto.Marshal(reg_node)
	if err != nil {
		xflog.ERROR("[register_myself]serilize name node error")
		return
	}
	res, err := zk.CreateNode(zk_node, bin_reg_node)
	if err != nil {
		xflog.ERROR("[register_myself]register node error", err)
		return
	} else {
		xflog.INFOF("[register_myself]complte register, %s", res)
		return
	}
}
