package mongodb

import (
	"optimusprime/optimusprime/log"
	"../../utils/mongodb/mgo"
	"sync"
)

const (
	MgoEventual  = 0
	MgoMonotonic = 1
	MgoStrong    = 2
)

var (
	MgoConnPoolMu sync.Mutex
	MgoConnPool   = make(map[string]*MgoConn)
)

type MgoConn struct {
	conn *mgo.Session
}

func initMgoConn(connstr string) (mgoConn *MgoConn, err error) {
	if connstr == "" {
		log.ERRORF("connect mongodb server failed:connection string is empty")
		return
	}
	session, err := mgo.Dial(connstr)
	if err != nil {
		log.ERRORF("connect mongodb server[\"%s\"],failed:%s", connstr, err)
		return
	}

	mgoConn = &MgoConn{
		conn: session,
	}

	mgoConn.SetMode(MgoMonotonic)

	MgoConnPoolMu.Lock()
	MgoConnPool[connstr] = mgoConn
	MgoConnPoolMu.Unlock()
	return
}

func GetMgoInstance(connstr string) (mgoConn *MgoConn, err error) {
	MgoConnPoolMu.Lock()
	conn, ok := MgoConnPool[connstr]
	MgoConnPoolMu.Unlock()
	if ok {
		mgoConn = conn
		return
	}
	return initMgoConn(connstr)
}

func (c *MgoConn) SetMode(mode int) {
	c.conn.SetMode(mgo.Mode(mode), true)
}

func (c *MgoConn) InsertDocs(db, collection string, docs []interface{}) (err error) {
	session := c.conn.Copy()
	defer session.Close()
	col := session.DB(db).C(collection)
	return col.Insert(docs...)
}

func (c *MgoConn) FindDocs(db, collection string, selector interface{}, limit int, result interface{}) (err error) {
	session := c.conn.Copy()
	defer session.Close()
	col := session.DB(db).C(collection)
	query1 := col.Find(selector)
	query2 := query1.Limit(limit)
	return query2.All(result)
}

func (c *MgoConn) FindDocsWithSort(db, collection string, selector interface{}, limit int, sortField string, result interface{}) (err error) {
	session := c.conn.Copy()
	defer session.Close()
	col := session.DB(db).C(collection)
	query1 := col.Find(selector).Sort(sortField)
	query2 := query1.Limit(limit)
	return query2.All(result)
}

func (c *MgoConn) UpdateDocs(db, collection string, selector interface{}, updateDoc interface{}) (err error) {
	session := c.conn.Copy()
	defer session.Close()
	col := session.DB(db).C(collection)
	_, err = col.UpdateAll(selector, updateDoc)
	return
}

func (c *MgoConn) UpsertDocs(db, collection string, selector interface{}, updateDoc interface{}) (err error) {
	session := c.conn.Copy()
	defer session.Close()
	col := session.DB(db).C(collection)
	_, err = col.Upsert(selector, updateDoc)
	return
}

func (c *MgoConn) RemoveDocs(db, collection string, selector interface{}) (err error) {
	session := c.conn.Copy()
	defer session.Close()
	col := session.DB(db).C(collection)
	_, err = col.RemoveAll(selector)
	return
}

func (c *MgoConn) CountDocs(db, collection string, selector interface{}) (num int, err error) {
	session := c.conn.Copy()
	defer session.Close()
	col := session.DB(db).C(collection)
	query1 := col.Find(selector)
	num, err = query1.Count()
	return
}

func (c *MgoConn) FindDocsWithSortSkip(db, collection string, selector interface{}, limit, offset int, sortField string, result interface{}) (err error) {
	session := c.conn.Copy()
	defer session.Close()
	col := session.DB(db).C(collection)
	query1 := col.Find(selector).Sort(sortField)
	query2 := query1.Skip(offset).Limit(limit)
	return query2.All(result)
}
