package utils

import (
	"github.com/panjf2000/ants/v2"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type timingWheelStatus int8

const (
	ready timingWheelStatus = iota
	running
	stopped
)

type TaskHandler func(data any, tc TaskContext)
type SubmitErrorHandler func(data any, err error)

func NewTimingWheel(intervalSeconds uint32, scale uint64) (instance *TimingWheel) {
	return newTimingWheel(intervalSeconds, scale, false, 0, nil, nil)
}
func NewTimingWheelWithPool(intervalSeconds uint32, scale uint64, poolSize int, submitErrorHandler SubmitErrorHandler, options ...ants.Option) (instance *TimingWheel) {
	return newTimingWheel(intervalSeconds, scale, true, poolSize, submitErrorHandler, options...)
}
func newTimingWheel(intervalSeconds uint32, scale uint64, usePool bool, poolSize int, submitErrorHandler SubmitErrorHandler, options ...ants.Option) (instance *TimingWheel) {
	instance = &TimingWheel{
		scale:    scale,
		interval: intervalSeconds,
		nodes:    make([]*node, scale),
		status:   ready,
	}
	if usePool {
		instance.submitErrHandler = submitErrorHandler
		pool, err := ants.NewPool(poolSize, options...)
		if err != nil {
			panic(err)
		}
		instance.pool = pool
	}
	head := &node{
		index: 0,
	}
	instance.nodes[0] = head
	tail := head
	for i := uint64(1); i < scale; i++ {
		n := &node{
			index: i,
		}
		tail.next = n
		tail = n
		instance.nodes[i] = tail
	}
	tail.next = head
	return
}

type task struct {
	round   uint64
	data    any
	handler TaskHandler
	tw      *TimingWheel
}

func (t *task) handle() {
	t.handler(t.data, t.tw)
}

type node struct {
	index uint64
	tasks []*task
	lock  sync.Mutex
	next  *node
}
type TimingWheel struct {
	interval         uint32
	scale            uint64
	nodes            []*node
	current          uint64
	stop             chan struct{}
	status           timingWheelStatus
	lock             sync.Mutex
	pool             *ants.Pool
	submitErrHandler SubmitErrorHandler
}

func (tw *TimingWheel) Start() {
	if tw.status == running {
		return
	}
	tw.lock.Lock()
	defer tw.lock.Unlock()
	tw.status = running
	tw.stop = make(chan struct{})
	go tw.run()
}
func (tw *TimingWheel) run() {
	ticker := time.NewTicker(time.Duration(tw.interval) * time.Second)
	for {
		select {
		case <-ticker.C:
			tw.tick()
		case <-tw.stop:
			ticker.Stop()
			log.Info("Timing wheel stopped!")
			return
		}
	}
}
func (tw *TimingWheel) tick() {
	tw.nodes[tw.current].lock.Lock()
	task2Handle := tw.nodes[tw.current].tasks
	tw.nodes[tw.current].tasks = nil
	tw.nodes[tw.current].lock.Unlock()
	for i := len(task2Handle) - 1; i >= 0; i-- {
		if task2Handle[i].round > 0 {
			task2Handle[i].round--
			continue
		}
		if tw.pool != nil {
			if err := tw.pool.Submit(func() {
				task2Handle[i].handle()
			}); err != nil {
				if tw.submitErrHandler != nil {
					tw.submitErrHandler(task2Handle[i].data, err)
				}
			}
		} else {
			go task2Handle[i].handle()
		}
		task2Handle = append(task2Handle[:i], task2Handle[i+1:]...)
	}
	tw.nodes[tw.current].lock.Lock()
	tw.nodes[tw.current].tasks = append(tw.nodes[tw.current].tasks, task2Handle...)
	tw.nodes[tw.current].lock.Unlock()
	tw.current = (tw.current + 1) % tw.scale
}
func (tw *TimingWheel) AddTask(data any, handler TaskHandler, duration time.Duration) {
	afterSeconds := uint64(duration.Seconds())
	//在当前node为开始往后面加任务时，多计算了一个刻度，要减去
	if afterSeconds >= uint64(tw.interval) {
		afterSeconds -= uint64(tw.interval)
	}
	index := (afterSeconds / uint64(tw.interval)) % tw.scale
	round := (afterSeconds / uint64(tw.interval)) / tw.scale
	t := &task{
		round:   round,
		data:    data,
		tw:      tw,
		handler: handler,
	}
	n := tw.nodes[(tw.current+index)%tw.scale]
	n.lock.Lock()
	defer n.lock.Unlock()
	n.tasks = append(n.tasks, t)
}
func (tw *TimingWheel) Stop() {
	if tw.status == running {
		tw.lock.Lock()
		if tw.status == running {
			close(tw.stop)
			tw.status = stopped
		}
		tw.lock.Unlock()
	}
}

type TaskContext interface {
	AddTask(data any, handler TaskHandler, duration time.Duration)
}
