/***************************************************
 * @Time : 2019/11/21 6:46 下午
 * @Author : ccoding
 * @File : rabbmitmq
 * @Software: GoLand
 **************************************************/
package pool

import (
	"container/list"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/streadway/amqp"
)
 
 var (
	 ErrInvalidConfig = errors.New("invalid pool config")
	 ErrPoolClosed    = errors.New("pool closed")
 )
 
 type PoolConfig struct {
	 MaxOpen     int           // 池中最大资源数
	 NumOpen     int           // 当前池中资源数
	 MinOpen     int           // 池中最少资源数
	 Closed      bool          // 池是否已关闭
	 Setintval 	 time.Duration //空闲连接连接超时时间
 }
 type NewConnection func() (*ObjMq, error)

 type ObjMq struct{
	Cn *amqp.Connection
	Ch *amqp.Channel
	Qu amqp.Queue
 }
 type RabbitmqPool struct {
	 mu    sync.Mutex
	 conns chan *ObjMq
	 newConnection func() (*ObjMq, error)
	 poolConfig    *PoolConfig
	 lq *list.List
 }
 
 func NewPool(config *PoolConfig, newConnection NewConnection) (*RabbitmqPool, error) {
	 if config.MaxOpen <= 0 || config.MinOpen > config.MaxOpen {
		 return nil, ErrInvalidConfig
	 }
	 p := &RabbitmqPool{
		conns:     		 make(chan *ObjMq, config.MaxOpen),
		newConnection: newConnection,
		poolConfig:    config,
		lq : list.New(),
	}

	 for i := 0; i < config.MinOpen; i++ {
		 cn, err := newConnection()
		 if err != nil {
			log.Printf(" error  %s", err)
			 continue
		 }
		 p.lq.PushBack(cn)
	 }
	 //检查离线deep
	 go p.AutoLink()
	 return p, nil
 }

 func (p *RabbitmqPool) AutoLink(){
	for{
		for i := 0; i < (p.poolConfig.MinOpen - p.lq.Len()  ); i++ {
			cn, err := p.newConnection()
			if err != nil {
			 log.Printf(" error  %s", err)
				continue
			}
			p.mu.Lock()
			p.lq.PushBack(cn)
			p.mu.Unlock()
		}
		time.Sleep(p.poolConfig.Setintval)
	}
	
 }
 func (p *RabbitmqPool) Get() (*ObjMq, error) {
	for ele := p.lq.Front();ele !=nil;ele = ele.Next(){
		
		if ele.Value.(*ObjMq).Cn.IsClosed(){
			p.Get() 
		}
		p.mu.Lock()
		p.lq.PushBack(ele.Value.(*ObjMq))
		p.mu.Unlock()
		return ele.Value.(*ObjMq) ,nil
	}
	return nil,errors.New("123123")
 }
 
 // 释放单个资源到连接池
 func (p *RabbitmqPool) Release(conn *ObjMq) error {
	//  if p.poolConfig.Closed {
	// 	 return ErrPoolClosed
	//  }
	//  p.conns <- conn
	 return nil
 }
 
 // 关闭单个资源
 func (p *RabbitmqPool) Close(conn *ObjMq) error {
	 conn.Cn.Close()
	 conn.Ch.Close()
	 p.poolConfig.NumOpen--
	 return nil
 }
 
 // 关闭连接池，释放所有资源
 func (p *RabbitmqPool) ClosePool() error {
	 if p.poolConfig.Closed {
		 return ErrPoolClosed
	 }
	 close(p.conns)
	 for conn := range p.conns {
		 conn.Cn.Close()
		 conn.Ch.Close()
		 p.poolConfig.NumOpen--
	 }
	 p.poolConfig.Closed = true
	 return nil
	}