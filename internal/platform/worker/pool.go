package worker

import (
	"context"
	"sync"
)

type Task func()

type Pool struct {
	workQueue chan Task
	wg        sync.WaitGroup
	workers   int
}

func NewPool(maxWorkers int) *Pool {
	return &Pool{
		workQueue: make(chan Task, maxWorkers*2),
		workers:   maxWorkers,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for {
				select {
				case task, ok := <-p.workQueue:
					if !ok {
						return
					}
					task()
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

func (p *Pool) Submit(task Task) {
	p.workQueue <- task
}

func (p *Pool) Stop() {
	close(p.workQueue)
	p.wg.Wait()
}
