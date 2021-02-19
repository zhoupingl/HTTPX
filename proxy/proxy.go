package proxy

import "sync"

type workers struct {
	bucket map[string]*worker
	sync.RWMutex
}

func NewWorkers() *workers {
	return &workers{
		bucket: map[string]*worker{},
	}
}

func (w *workers) GetWorker(url string) *worker {
	p := w.GetWorker1(url)
	if p == nil {
		p = w.GetWorker2(url)
	}

	return p
}

// 使用读锁 访问数据
func (w *workers) GetWorker1(url string) *worker {
	w.RLock()
	defer w.RUnlock()

	if p, ok := w.bucket[url]; ok {
		return p
	}

	return nil
}

// 写锁访问数据，如果在读锁 获取数据未成功。则使用写锁读取更新数据
func (w *workers) GetWorker2(url string) *worker {
	w.Lock()
	defer w.Unlock()

	p, ok := w.bucket[url]
	if !ok {
		p = NewWorker(100)
		w.bucket[url] = p
	}

	return p
}
