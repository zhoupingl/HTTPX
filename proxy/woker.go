package proxy

import (
	"time"
)

type worker struct {
	work chan func()
	sem  chan uint8
	q    chan func()
}

func NewWorker(size int) *worker {
	p := &worker{
		work: make(chan func()),
		q:    make(chan func(), 1024),
		sem:  make(chan uint8, size),
	}

	go p.backgroundSchedule()

	return p
}

func (p *worker) Schedule(task func()) {
	p.q <- task
}

func (p *worker) backgroundSchedule() {

	for task := range p.q {
		select {
		case p.work <- task:
		case p.sem <- 1:
			go p.worker(task)
		}
	}

}

func (p *worker) worker(task func()) {
	defer func() { <-p.sem }()
	for {
		task()
		select {
		case task = <-p.work:
			// 长时间没有任务，关闭协程
		case <-time.After(time.Hour):
			return
		}
	}
}
