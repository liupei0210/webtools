package utils

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

func NewTimingWheel(intervalSeconds uint32, scale uint64) (instance *TimingWheel) {
	instance = &TimingWheel{
		scale:    scale,
		interval: intervalSeconds,
		nodes:    make([]*node, scale),
		stop:     make(chan struct{}),
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
	round uint64
	data  any
}
type node struct {
	index uint64
	tasks []*task
	lock  sync.Mutex
	next  *node
}
type TimingWheel struct {
	interval uint32
	scale    uint64
	nodes    []*node
	current  uint64
	stop     chan struct{}
}

func (tw *TimingWheel) Start(process func(any)) {
	go func() {
		ticker := time.NewTicker(time.Duration(tw.interval) * time.Second)
		for {
			select {
			case <-ticker.C:
				tw.nodes[tw.current].lock.Lock()
				var newTasks []*task
				for _, t := range tw.nodes[tw.current].tasks {
					if t.round > 0 {
						t.round -= 1
						newTasks = append(newTasks, t)
						continue
					}
					go process(t.data)
				}
				tw.nodes[tw.current].tasks = newTasks
				tw.nodes[tw.current].lock.Unlock()
				tw.current = (tw.current + 1) % tw.scale
			case <-tw.stop:
				ticker.Stop()
				log.Info("Timing wheel stopped!")
				return
			}
		}
	}()
}
func (tw *TimingWheel) AddTask(data any, duration time.Duration) {
	afterSeconds := uint64(duration.Seconds())
	//在当前node为开始往后面加任务时，多计算了一个刻度，要减去
	if afterSeconds >= uint64(tw.interval) {
		afterSeconds -= uint64(tw.interval)
	}
	index := (afterSeconds / uint64(tw.interval)) % tw.scale
	round := (afterSeconds / uint64(tw.interval)) / tw.scale
	t := &task{
		round: round,
		data:  data,
	}
	n := tw.nodes[(tw.current+index)%tw.scale]
	n.lock.Lock()
	defer n.lock.Unlock()
	n.tasks = append(n.tasks, t)
}
func (tw *TimingWheel) Stop() {
	close(tw.stop)
}
