package core

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

// WaitGroup is a wrapper for sync.WaitGroup
type WaitGroup struct {
	sync.WaitGroup
}

// Run runs a function in a goroutine, adds 1 to WaitGroup
// and calls done when function returns. Please DO NOT use panic
// in the cb function.
func (w *WaitGroup) RunWith(exec func()) {
	w.Add(1)
	go func() {
		defer w.Done()
		exec()
	}()
}

// RunWithRecover wraps goroutine startup call with force recovery, add 1 to WaitGroup
// and call done when function return. it will dump current goroutine stack into log if catch any recover result.
// exec is that execute logic function. recoverFn is that handler will be called after recover and before dump stack,
// passing `nil` means noop.
func (w *WaitGroup) RunWithRecover(exec func(), recoverFn func(r interface{})) {
	w.Add(1)
	go func() {
		defer func() {
			r := recover()
			if r != nil && recoverFn != nil {
				stack := debug.Stack()
				log.Printf("panic: %v\n%s", r, stack)
				recoverFn(r)
			}
			w.Done()
		}()
		exec()
	}()
}

func (w *WaitGroup) RunRecover(exec func(v interface{}), recoverFn func(r interface{}), v interface{}) {
	//w.Add(1)
	go func() {
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
			//w.Done()
		}()
		exec(v)
	}()
}

func (w *WaitGroup) Wait() {
	w.Wait()
}
