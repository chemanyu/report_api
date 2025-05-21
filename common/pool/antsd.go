package pool

import (
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/panjf2000/ants/v2"
)

type GoroutinePool struct {
	pool *ants.Pool
	wg   sync.WaitGroup
}

var Goroutine *GoroutinePool

func init() {
	Goroutine, _ = NewGoroutinePool(20000)
}

func NewGoroutinePool(size int) (*GoroutinePool, error) {
	p, err := ants.NewPool(size)
	if err != nil {
		return nil, err
	}
	return &GoroutinePool{
		pool: p,
	}, nil
}

func (gp *GoroutinePool) Submit(task func(v interface{}), recoverFn func(r any), v any) error {
	//gp.wg.Add(1)
	return gp.pool.Submit(func() {
		defer func() {
			r := recover()
			if r != nil && recoverFn != nil {
				debug.PrintStack()
				var errStr string
				if err, ok := r.(error); ok {
					errStr = err.Error()
				} else {
					errStr = fmt.Sprint(r)
				}
				recoverFn(fmt.Sprintf("Recovered from panic: %v\n", errStr))
			}
		}()
		task(v)
		//gp.wg.Done()
	})
}

func (gp *GoroutinePool) Wait() {
	gp.wg.Wait()
}

func (gp *GoroutinePool) Release() {
	gp.pool.Release()
}
