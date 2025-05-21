package core

import (
	"database/sql"
	"log"
	"sync"

	"golang.org/x/sync/singleflight"

	"github.com/garyburd/redigo/redis"
	"github.com/gin-gonic/gin"
	"github.com/golang/groupcache/lru"
	jsoniter "github.com/json-iterator/go"
)

var sg singleflight.Group

type Mate struct {
	lruData   *lru.Cache
	redisPool *redis.Pool
	mysqlPool *sql.DB
	pushLock  *sync.RWMutex
	hash      Hasher
}

type CallbackConf struct {
	Upstream   string `json:"upstream"`
	Downstream string `json:"downstream"`
	Rate       int    `json:"rate"`
	IsMaster   bool   `json:"is_master"`
}

func NewData(rdPool *redis.Pool, msPool *sql.DB) *Mate {
	return &Mate{
		lruData:   lru.New(100000),
		redisPool: rdPool,
		mysqlPool: msPool,
		hash:      newDefaultHasher(),
		pushLock:  new(sync.RWMutex),
	}
}

type mateData map[string]interface{}

// 查找获取缓存
func (c *Mate) Get(key string) map[string]interface{} {
	key = "ocpx-unikey:" + key

	// result, ok := c.lruData.Get(key)
	// if !ok {
	// 	result = c.redisGet(key)
	// }

	//没用sg测试
	// var result interface{}
	// result = c.redisGet(key)
	// ret := mateData{}
	// if result != nil {

	// 	err := jsoniter.Unmarshal(result.([]byte), &ret)
	// 	if err != nil {
	// 		if gin.DebugMode == "debug" {
	// 			log.Println(err.Error(), string(result.([]byte)))
	// 		}
	// 	}
	// } else {
	// 	return nil
	// }
	// return ret
	v, _, _ := sg.Do(key, func() (interface{}, error) {
		var result interface{}
		result = c.redisGet(key)
		ret := mateData{}
		if result != nil {

			err := jsoniter.Unmarshal(result.([]byte), &ret)
			if err != nil {
				if gin.DebugMode == "debug" {
					//log.Println(err.Error(), string(result.([]byte)))
				}
			}
		} else {
			return nil, nil
		}
		//拦截并发
		//fmt.Println(1)
		return ret, nil
	})
	//复用
	//fmt.Println(2)
	if v != nil {
		ret, _ := v.(mateData)

		return ret
	}
	return nil
}

func (c *Mate) lruGet(key string) (ret []byte) {
	c.pushLock.RLock()
	tmpRet, ok := c.lruData.Get(key)
	c.pushLock.RUnlock()
	if ok {
		return tmpRet.([]byte)
	} else {
		return nil
	}
}

func (c *Mate) redisGet(key string) (ret []byte) {
	r := c.redisPool.Get()
	defer r.Close()
	ret, err := redis.Bytes(r.Do("GET", key))
	if err != nil {
		if gin.DebugMode == "debug" {
			log.Println("redisGet:", err.Error())
		}
	}
	if ret != nil {
		c.pushLock.Lock()
		c.lruData.Add(key, ret)
		c.pushLock.Unlock()
	}
	return ret
}

func (c *Mate) PushRedisAll() error {
	ret, _ := c.getAll("select main.*,ad.advertiser_id,ad.agent_id,ad.support_id,p.channel as product_channel,p.event_key,p.form as product_form,d.form as channel_form from ad_tool_monitor as main left join ad_tool_product as p on main.product_id = p.id left join ad_tool_channel as d on main.channel_id = d.id left join ad_tool_advertiser as ad on ad.advertiser_id = main.advertiser_id where main.del_flag=0")
	r := c.redisPool.Get()
	defer r.Close()
	for _, v := range ret {
		keyName := "ocpx-unikey:" + v["unikey"].(string)
		retString, _ := jsoniter.Marshal(v)
		c.setCaches(r, keyName, retString)
	}
	return nil
}

func (c *Mate) setCaches(r redis.Conn, key string, retString []byte) {
	r.Do("SET", key, retString)
	r.Do("EXPIRE", key, 86400)
}

func (c *Mate) scanRow(rows *sql.Rows) (a mateData, er error) {
	columns, _ := rows.Columns()

	vals := make([]interface{}, len(columns))
	valsPtr := make([]interface{}, len(columns))

	for i := range vals {
		valsPtr[i] = &vals[i]
	}

	err := rows.Scan(valsPtr...)

	if err != nil {
		if gin.DebugMode == "debug" {
			log.Println("scan err", err.Error())
		}
		return nil, err
	}

	r := make(mateData)

	for i, v := range columns {
		if va, ok := vals[i].([]byte); ok {
			r[v] = string(va)
		} else {
			r[v] = vals[i]
		}
	}

	return r, nil

}

// 获取一行记录
func (c *Mate) getOne(sql string, args ...interface{}) (mateData, error) {
	rows, err := c.mysqlPool.Query(sql, args...)
	if err != nil {
		if gin.DebugMode == "debug" {
			log.Println(11111111, err.Error())
		}
		return nil, err
	}
	rows.Next()
	defer rows.Close()

	result, err := c.scanRow(rows)
	return result, err
}

// 获取多行记录
func (c *Mate) getAll(sql string, args ...interface{}) ([]mateData, error) {
	rows, err := c.mysqlPool.Query(sql, args...)
	if err != nil {
		if gin.DebugMode == "debug" {
			log.Println(11111, sql, err.Error())
		}
		return nil, err
	}

	defer rows.Close()

	result := make([]mateData, 0)

	for rows.Next() {
		r, err := c.scanRow(rows)
		if err != nil {
			if gin.DebugMode == "debug" {
				log.Println(11111, err.Error())
			}
			continue
		}

		result = append(result, r)
	}

	return result, nil

}
