/*=============================================================================
#     FileName: redis.go
#         Desc:  redis operations's wrap
#       Author: ato.ye
#        Email: ato.ye@ucloud.cn
#     HomePage: http://www.ucloud.cn
#      Version: 0.0.1
#   LastChange: 2016-04-20 18:17:35
#      History:
=============================================================================*/
package redis

import (
	"../log"
	"../utils/redis/redis"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	redisConnPoolMu sync.Mutex
	redisConnPool   = make(map[string]*RedisConn)
)

type RedisConn struct {
	conn *redis.Client
}

func initRedisConn(connStr string, dialTimeout time.Duration) (redisConn *RedisConn, err error) {
	var errStr string
	if connStr == "" {
		errStr = "connect redis server failed: connection string is empty"
		xflog.ERRORF(errStr)
		err = errors.New(errStr)
		return
	}

	client := redis.NewClient(&redis.Options{
		Addr:        connStr,
		Password:    "",
		DB:          0,
		DialTimeout: dialTimeout,
	})

	if client == nil {
		errStr = fmt.Sprintf("new redis client[%s] failed", connStr)
		xflog.ERRORF(errStr)
		err = errors.New(errStr)
		return
	}
	_, err = client.Ping().Result()
	if err != nil {
		xflog.ERRORF("connect to [%s] faild:%v", connStr, err)
		return
	}

	redisConn = &RedisConn{
		conn: client,
	}

	redisConnPoolMu.Lock()
	redisConnPool[connStr] = redisConn
	redisConnPoolMu.Unlock()
	return
}

func GetRedisInstance(connStr string) (redisConn *RedisConn, err error) {
	redisConnPoolMu.Lock()
	conn, ok := redisConnPool[connStr]
	redisConnPoolMu.Unlock()
	if ok {
		redisConn = conn
		return
	}
	return initRedisConn(connStr, 5*time.Second)
}

func GetRedisInstanceWithOptions(connStr string, dialTimeout time.Duration) (
	redisConn *RedisConn, err error) {
	redisConnPoolMu.Lock()
	conn, ok := redisConnPool[connStr]
	redisConnPoolMu.Unlock()
	if ok {
		redisConn = conn
		return
	}
	return initRedisConn(connStr, dialTimeout)
}

func (c *RedisConn) HMSet(key, field, value string, pairs ...string) error {
	status := c.conn.HMSet(key, field, value, pairs...)
	return status.Err()
}

func (c *RedisConn) HMGet(key string, fields ...string) (interface{}, error) {
	slice := c.conn.HMGet(key, fields...)
	content, err := slice.Result()
	return content, err
}

func (c *RedisConn) HMDelete(key string, fields ...string) error {
	status := c.conn.HDel(key, fields...)
	return status.Err()
}

func (c *RedisConn) HMExists(key, field string) error {
	status := c.conn.HExists(key, field)
	return status.Err()
}

func (c *RedisConn) Exists(key string) bool {
	is_exists := c.conn.Exists(key)
	return is_exists.Val()
}

// keys op
func (c *RedisConn) Expire(key string, expiration time.Duration) (bool, error) {
	status := c.conn.Expire(key, expiration)
	return status.Result()
}

func (c *RedisConn) Rename(key, newKey string) error {
	status := c.conn.Rename(key, newKey)
	return status.Err()
}

func (c *RedisConn) Keys(pattern string) ([]string, error) {
	status := c.conn.Keys(pattern)
	return status.Result()
}

func (c *RedisConn) KeyExists(key string) (bool, error) {
	status := c.conn.Exists(key)
	return status.Result()
}

func (c *RedisConn) Del(key string) error {
	status := c.conn.Del(key)
	return status.Err()
}

func (c *RedisConn) Set(key string, value string, expir time.Duration) error {
	status := c.conn.Set(key, value, expir)
	return status.Err()
}

func (c *RedisConn) Get(key string) ([]byte, error) {
	status := c.conn.Get(key)
	b, _ := status.Bytes()
	return b, status.Err()
}

func (c *RedisConn) HGetAll(key string) (map[string]string, error) {
	status := c.conn.HGetAllMap(key)
	return status.Result()
}

// redis list op
func (c *RedisConn) LPop(key string) (string, error) {
	status := c.conn.LPop(key)
	return status.Result()
}

func (c *RedisConn) RPush(key string, values ...string) (int64, error) {
	status := c.conn.RPush(key, values...)
	return status.Result()
}

func (c *RedisConn) LLen(key string) (int64, error) {
	status := c.conn.LLen(key)
	return status.Result()
}

// redis sorted set op
func (c *RedisConn) ZInterStore(dest string, aggregate string, keys ...string) error {
	weights := make([]float64, len(keys))
	for i, n := 0, len(keys); i < n; i++ {
		weights[i] = 1
	}
	t := redis.ZStore{
		weights,
		aggregate,
	}
	status := c.conn.ZInterStore(dest, t, keys...)
	return status.Err()
}

func (c *RedisConn) ZCard(key string) (int64, error) {
	status := c.conn.ZCard(key)
	return status.Result()
}

func (c *RedisConn) ZRange(key string, start, stop int64) ([]string, error) {
	status := c.conn.ZRange(key, start, stop)
	return status.Result()
}

func (c *RedisConn) ZRevRange(key string, start, stop int64) ([]string, error) {
	status := c.conn.ZRevRange(key, start, stop)
	return status.Result()
}

func (c *RedisConn) ZAddBatch(key string, score_lst []float64, data_lst []interface{}) int64 {
	mem_lst := make([]redis.Z, 0)
	for i := 0; i < len(score_lst); i++ {
		tmp := redis.Z{
			Score:  score_lst[i],
			Member: data_lst[i],
		}

		mem_lst = append(mem_lst, tmp)
	}
	status := c.conn.ZAdd(key, mem_lst...)
	return status.Val()
}

func (c *RedisConn) ZRangeByScore(key, min, max string, offset, count int64) ([]string, error) {
	opt := &redis.ZRangeByScore{
		Min:    min,
		Max:    max,
		Offset: offset,
		Count:  count,
	}
	status := c.conn.ZRangeByScore(key, *opt)
	return status.Result()
}

func (c *RedisConn) ZRangeByScoreWithScores(key, min, max string, offset, count int64) ([]redis.Z, error) {
	opt := &redis.ZRangeByScore{
		Min:    min,
		Max:    max,
		Offset: offset,
		Count:  count,
	}
	status := c.conn.ZRangeByScoreWithScores(key, *opt)
	return status.Result()
}
