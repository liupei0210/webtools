package utils

import (
	"fmt"
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

// TaskHandler 任务处理函数
type TaskHandler func(data any, tc TaskContext)

// SubmitErrorHandler 任务提交错误处理函数
type SubmitErrorHandler func(data any, err error)

// TimingWheelOption 时间轮配置选项
type TimingWheelOption func(*TimingWheel)

// WithPoolSize 设置协程池大小
func WithPoolSize(size int) TimingWheelOption {
	return func(tw *TimingWheel) {
		tw.poolSize = size
	}
}

// WithErrorHandler 设置错误处理器
func WithErrorHandler(handler SubmitErrorHandler) TimingWheelOption {
	return func(tw *TimingWheel) {
		tw.submitErrHandler = handler
	}
}

// WithPoolOptions 设置协程池选项
func WithPoolOptions(options ...ants.Option) TimingWheelOption {
	return func(tw *TimingWheel) {
		tw.poolOptions = options
	}
}

// WithMaxTasksPerSlot 设置每个槽位最大任务数
func WithMaxTasksPerSlot(max int) TimingWheelOption {
	return func(tw *TimingWheel) {
		tw.maxTasksPerSlot = max
	}
}

// WithMetrics 设置是否启用指标收集
func WithMetrics(enable bool) TimingWheelOption {
	return func(tw *TimingWheel) {
		tw.enableMetrics = enable
	}
}

// NewTimingWheel 创建一个新的时间轮，不使用协程池
func NewTimingWheel(intervalSeconds uint32, scale uint64) *TimingWheel {
	return newTimingWheel(intervalSeconds, scale, false, nil)
}

// NewTimingWheelWithPool 创建一个新的时间轮，使用协程池
func NewTimingWheelWithPool(intervalSeconds uint32, scale uint64, opts ...TimingWheelOption) *TimingWheel {
	return newTimingWheel(intervalSeconds, scale, true, opts)
}

func newTimingWheel(intervalSeconds uint32, scale uint64, usePool bool, opts []TimingWheelOption) *TimingWheel {
	if intervalSeconds == 0 || scale == 0 {
		panic("interval and scale must be greater than 0")
	}

	tw := &TimingWheel{
		scale:           scale,
		interval:        intervalSeconds,
		nodes:           make([]*node, scale),
		status:          ready,
		usePool:         usePool,
		taskQueue:       make(chan *task, 1000),
		maxTasksPerSlot: 10000, // 默认每个槽位最大任务数
	}

	if tw.enableMetrics {
		tw.metrics = &Metrics{}
	}

	if usePool && opts != nil {
		for _, opt := range opts {
			opt(tw)
		}
	}

	tw.initNodes()
	return tw
}

type task struct {
	round   uint64
	data    any
	handler TaskHandler
	tw      *TimingWheel
}

func (t *task) handle() {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("任务处理发生panic: %v", r)
		}
	}()
	t.handler(t.data, t.tw)
}

type node struct {
	index uint64
	tasks []*task
	lock  sync.Mutex
	next  *node
}

// TimingWheel 时间轮实现
type TimingWheel struct {
	interval         uint32
	scale            uint64
	nodes            []*node
	current          uint64
	stop             chan struct{}
	status           timingWheelStatus
	lock             sync.Mutex
	pool             *ants.Pool
	usePool          bool
	poolSize         int
	poolOptions      []ants.Option
	submitErrHandler SubmitErrorHandler
	maxTasksPerSlot  int
	enableMetrics    bool
	metrics          *Metrics
	taskQueue        chan *task // 用于任务缓冲
}

func (tw *TimingWheel) initNodes() {
	head := &node{index: 0}
	tw.nodes[0] = head
	tail := head

	for i := uint64(1); i < tw.scale; i++ {
		n := &node{index: i}
		tail.next = n
		tail = n
		tw.nodes[i] = tail
	}
	tail.next = head
}

// Start 启动时间轮
func (tw *TimingWheel) Start() error {
	tw.lock.Lock()
	defer tw.lock.Unlock()

	if tw.status == running {
		return fmt.Errorf("时间轮已经在运行")
	}

	if tw.usePool {
		if tw.poolSize == 0 {
			tw.poolSize = 1000 // 默认池大小
		}
		var err error
		tw.pool, err = ants.NewPool(tw.poolSize, tw.poolOptions...)
		if err != nil {
			return fmt.Errorf("创建协程池失败: %v", err)
		}
	}

	tw.status = running
	tw.stop = make(chan struct{})
	go tw.run()
	return nil
}

func (tw *TimingWheel) run() {
	ticker := time.NewTicker(time.Duration(tw.interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			tw.tick()
		case <-tw.stop:
			if tw.pool != nil {
				tw.pool.Release()
			}
			log.Info("时间轮已停止!")
			return
		}
	}
}

func (tw *TimingWheel) tick() {
	currentNode := tw.nodes[tw.current]
	currentNode.lock.Lock()
	tasks := currentNode.tasks
	currentNode.tasks = nil
	currentNode.lock.Unlock()

	if len(tasks) == 0 {
		tw.current = (tw.current + 1) % tw.scale
		return
	}

	// 使用任务队列进行缓冲
	go func() {
		for _, task := range tasks {
			if task.round > 0 {
				task.round--
				tw.reinsertTask(task)
				continue
			}
			tw.processTask(task)
		}
	}()

	tw.current = (tw.current + 1) % tw.scale
}

// AddTask 添加定时任务
func (tw *TimingWheel) AddTask(data any, handler TaskHandler, duration time.Duration) error {
	if tw.status != running {
		return fmt.Errorf("时间轮未启动")
	}

	if duration < 0 {
		return fmt.Errorf("duration不能为负数")
	}

	afterSeconds := uint64(duration.Seconds())
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

	node := tw.nodes[(tw.current+index)%tw.scale]
	node.lock.Lock()
	node.tasks = append(node.tasks, t)
	node.lock.Unlock()

	return nil
}

// Stop 停止时间轮
func (tw *TimingWheel) Stop() {
	tw.lock.Lock()
	defer tw.lock.Unlock()

	if tw.status == running {
		close(tw.stop)
		tw.status = stopped
	}
}

// TaskContext 任务上下文接口
type TaskContext interface {
	AddTask(data any, handler TaskHandler, duration time.Duration) error
}

// GetMetrics 获取指标数据
func (tw *TimingWheel) GetMetrics() *Metrics {
	if !tw.enableMetrics {
		return nil
	}
	return tw.metrics
}

// CancelTask 取消任务
func (tw *TimingWheel) CancelTask(taskID string) error {
	// 实现任务取消逻辑
	return nil
}

// UpdateTask 更新任务执行时间
func (tw *TimingWheel) UpdateTask(taskID string, newDuration time.Duration) error {
	// 实现任务更新逻辑
	return nil
}

func (tw *TimingWheel) processTask(t *task) {
	if tw.enableMetrics {
		tw.metrics.mutex.Lock()
		tw.metrics.ProcessingTasks++
		tw.metrics.mutex.Unlock()
	}

	if tw.pool != nil {
		if err := tw.pool.Submit(func() {
			defer tw.taskCompleted(t)
			t.handle()
		}); err != nil {
			tw.handleTaskError(t, err)
		}
	} else {
		go func() {
			defer tw.taskCompleted(t)
			t.handle()
		}()
	}
}

func (tw *TimingWheel) taskCompleted(t *task) {
	if !tw.enableMetrics {
		return
	}
	tw.metrics.mutex.Lock()
	tw.metrics.CompletedTasks++
	tw.metrics.ProcessingTasks--
	tw.metrics.mutex.Unlock()
}

func (tw *TimingWheel) handleTaskError(t *task, err error) {
	if tw.enableMetrics {
		tw.metrics.mutex.Lock()
		tw.metrics.FailedTasks++
		tw.metrics.ProcessingTasks--
		tw.metrics.mutex.Unlock()
	}

	if tw.submitErrHandler != nil {
		tw.submitErrHandler(t.data, err)
	} else {
		log.Errorf("提交任务到协程池失败: %v", err)
	}
}

func (tw *TimingWheel) reinsertTask(t *task) {
	node := tw.nodes[(tw.current+1)%tw.scale]
	node.lock.Lock()
	if tw.maxTasksPerSlot > 0 && len(node.tasks) >= tw.maxTasksPerSlot {
		log.Warnf("槽位任务数超过限制: %d", tw.maxTasksPerSlot)
	}
	node.tasks = append(node.tasks, t)
	node.lock.Unlock()
}

// 添加指标结构
type Metrics struct {
	TotalTasks      int64
	CompletedTasks  int64
	FailedTasks     int64
	ProcessingTasks int64
	mutex           sync.RWMutex
}
