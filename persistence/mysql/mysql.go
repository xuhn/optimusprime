package mysql

import (
	"optimusprime/common"
	"optimusprime/log"
	_ "github.com/go-sql-driver/mysql"
	"database/sql"
	"errors"
	"sync"
)

const (
	NON_TRANSACTION string = ""
)

type MysqlConn struct {
	db     *sql.DB
	tx_map map[string]*sql.Tx
}

var (
	mysqlConnPoolMu sync.Mutex
	mysqlConnPool   = make(map[string]*MysqlConn)
)

func initMysqlConn(connStr string) (mysqlConn *MysqlConn, err error) {
	db, err := sql.Open("mysql", connStr)
	if err != nil {
		log.ERRORF("open mysql connection[%s] fail:%v", connStr, err)
		return
	}
	// 实际连接一次, 判断连接是否成功
	err = db.Ping()
	if err != nil {
		log.ERRORF("connect to [%s] fail:%v", connStr, err)
		return
	}

	mysqlConn = &MysqlConn{
		db:     db,
		tx_map: make(map[string]*sql.Tx, 0),
	}
	mysqlConnPoolMu.Lock()
	mysqlConnPool[connStr] = mysqlConn
	mysqlConnPoolMu.Unlock()
	return
}

// 获取mysql连接实例
func GetMysqlInstance(connStr string) (mysqlConn *MysqlConn, err error) {
	mysqlConnPoolMu.Lock()
	conn, ok := mysqlConnPool[connStr]
	mysqlConnPoolMu.Unlock()
	if ok {
		mysqlConn = conn
		return
	}
	return initMysqlConn(connStr)
}

// 新增
func (c *MysqlConn) Insert(sql string, args ...interface{}) (lastInsertId int64, err error) {
	res, err := c.db.Exec(sql, args...)
	if err != nil {
		log.ERRORF("exec insert sql[%s] fail:%v", sql, err)
		return
	}
	return res.LastInsertId()
}

// Tx新增
func (c *MysqlConn) TxInsert(txId, sql string, args ...interface{}) (lastInsertId int64, err error) {
	if txId == NON_TRANSACTION {
		return c.Insert(sql, args...)
	}
	tx, ok := c.tx_map[txId]
	if !ok {
		log.ERRORF("tx_id[%s] not exist.", txId)
		err = errors.New("tx_id not exist, failed to create transaction.")
		return
	}
	res, err := tx.Exec(sql, args...)
	if err != nil {
		log.ERRORF("exec insert sql[%s] fail:%v", sql, err)
		return
	}
	return res.LastInsertId()
}

// 修改
func (c *MysqlConn) Update(sql string, args ...interface{}) (rowsAffected int64, err error) {
	res, err := c.db.Exec(sql, args...)
	if err != nil {
		log.ERRORF("exec update sql[%s] fail:%v", sql, err)
		return
	}
	return res.RowsAffected()
}

// Tx修改
func (c *MysqlConn) TxUpdate(txId, sql string, args ...interface{}) (lastInsertId int64, err error) {
	if txId == NON_TRANSACTION {
		return c.Update(sql, args...)
	}
	tx, ok := c.tx_map[txId]
	if !ok {
		log.ERRORF("tx_id[%s] not exist.", txId)
		err = errors.New("tx_id not exist, failed to create transaction.")
		return
	}
	res, err := tx.Exec(sql, args...)
	if err != nil {
		log.ERRORF("exec update sql[%s] fail:%v", sql, err)
		return
	}
	return res.RowsAffected()
}

// 删除
func (c *MysqlConn) Delete(sql string, args ...interface{}) (rowsAffected int64, err error) {
	res, err := c.db.Exec(sql, args...)
	if err != nil {
		log.ERRORF("exec delete sql[%s] fail:%v", sql, err)
		return
	}
	return res.RowsAffected()
}

// Tx修改
func (c *MysqlConn) TxDelete(txId, sql string, args ...interface{}) (lastInsertId int64, err error) {
	if txId == NON_TRANSACTION {
		return c.Delete(sql, args...)
	}
	tx, ok := c.tx_map[txId]
	if !ok {
		log.ERRORF("tx_id[%s] not exist.", txId)
		err = errors.New("tx_id not exist, failed to create transaction.")
		return
	}
	res, err := tx.Exec(sql, args...)
	if err != nil {
		log.ERRORF("exec delete sql[%s] fail:%v", sql, err)
		return
	}
	return res.RowsAffected()
}

// 查询
func (c *MysqlConn) Select(sql string, args ...interface{}) (results []map[string]string, err error) {
	rows, err := c.db.Query(sql, args...)
	if err != nil {
		return
	}
	// 关闭数据集
	defer rows.Close()
	// 列信息
	columns, _ := rows.Columns()
	// 列值
	values := make([][]byte, len(columns))
	// 扫描器
	scans := make([]interface{}, len(columns))
	for i := range values {
		scans[i] = &values[i]
	}
	results = make([]map[string]string, 0)

	for rows.Next() {
		if err = rows.Scan(scans...); err != nil {
			log.ERRORF("exec select sql[%s] fail:%v", sql, err)
			return
		}
		row := make(map[string]string)
		for k, v := range values {
			key := columns[k]
			row[key] = string(v)
		}
		results = append(results, row)
	}
	return
}

//创建Transaction
func (c *MysqlConn) BeginTx() (txId string, err error) {
	txId = common.NewUUIDV4().String()
	if _, ok := c.tx_map[txId]; ok {
		log.ERRORF("tx_id[%s] exist.", txId)
		err = errors.New("tx_id exist, failed to create transaction.")
		return
	}
	c.tx_map[txId], err = c.db.Begin()
	log.DEBUGF("BeginTx[%v]", c.tx_map)
	if err != nil {
		log.ERRORF("create transaction[%s] error.", txId)
		err = errors.New("failed to create transaction.")
		return
	}
	return
}

//Commit
func (c *MysqlConn) Commit(txId string) (err error) {
	tx, ok := c.tx_map[txId]
	if !ok {
		log.ERRORF("tx_id[%s] not exist.", txId)
		err = errors.New("tx_id not exist, failed to commit transaction.")
		return
	}
	log.DEBUGF("tx_id[%s] begin to commit.", txId)
	defer delete(c.tx_map, txId)
	if err = tx.Commit(); err != nil {
		tx.Rollback() // 错误回滚
		log.ERRORF("tx_id[%s] commit error: %v", txId, err)
		return
	}
	return
}

//Commit
func (c *MysqlConn) Rollback(txId string) (err error) {
	tx, ok := c.tx_map[txId]
	if !ok {
		err = errors.New("tx_id not exist, failed to rollback transaction.")
		return
	}
	log.DEBUGF("tx_id[%s] begin to rollback.", txId)
	err = tx.Rollback() // 错误回滚
	if err != nil {
		log.ERRORF("Rollback:err[%v]", err)
	}
	delete(c.tx_map, txId)
	return
}
