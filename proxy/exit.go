package proxy

import (
	"go.uber.org/atomic"
	"time"
)

var _exit = NewExit()

type exit struct {
	stat  *atomic.Int64
	_exit *atomic.Bool
}

func Exit() {
	_exit.Exit()
}

func Wait() {
	_exit.Wait()
}

func NewExit() *exit {
	_exit := &exit{
		stat:  atomic.NewInt64(0),
		_exit: atomic.NewBool(false),
	}
	_exit.run()

	return _exit

}

func (e *exit) Add(i int64) {
	e.stat.Add(i)
}

func (e *exit) IsRun() bool {
	return e._exit.Load()
}
func (e *exit) run() {
	e._exit.Store(true)
}

func (e *exit) Exit() {
	e._exit.Store(false)
}

func (e *exit) Done() {
	e.stat.Sub(1)
}

func (e *exit) Wait() {
	for {
		if e.stat.Load() == 0 {
			return
		}
		time.Sleep(time.Second)
	}
}
