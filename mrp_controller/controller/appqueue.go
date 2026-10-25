package controller

import (
	"container/list"
	"context"
	"sync"

	log "github.com/sirupsen/logrus"

	appv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
)

type AppQueueItem struct {
	Key   string
	Value appv1.Application
}

type gaugeSetter interface {
	Set(float64)
}

type KeyedAppQueue struct {
	mu     sync.RWMutex
	items  *list.List
	keyMap map[string]*list.Element
	// currently handled keys
	processing map[string]struct{}
	// updates received during keys processing, re-queued by Done
	pending     map[string]appv1.Application
	notEmpty    *sync.Cond
	lengthGauge gaugeSetter
}

func NewKeyedAppQueue() *KeyedAppQueue {
	kaq := &KeyedAppQueue{
		items:      list.New(),
		keyMap:     make(map[string]*list.Element),
		processing: make(map[string]struct{}),
		pending:    make(map[string]appv1.Application),
	}
	kaq.notEmpty = sync.NewCond(&kaq.mu)
	return kaq
}

func (kq *KeyedAppQueue) Enqueue(app appv1.Application) {
	kq.mu.Lock()
	defer kq.mu.Unlock()
	kq.enqueueLocked(app)
}

func (kq *KeyedAppQueue) enqueueLocked(app appv1.Application) {
	key := app.Namespace + "/" + app.Name

	// put into pending list if it is currently being processed
	if _, processing := kq.processing[key]; processing {
		kq.pending[key] = app
		kq.reportQueueLength()
		return
	}

	item := AppQueueItem{Key: key, Value: app}
	var newElem *list.Element
	if elem, exists := kq.keyMap[key]; exists {
		// put the new variant of same app at the same place in queue
		newElem = kq.items.InsertAfter(item, elem)
		kq.items.Remove(elem)
	} else {
		// add new app to the end of the queue
		newElem = kq.items.PushBack(item)
	}
	kq.keyMap[key] = newElem
	// signal workers they got job
	kq.notEmpty.Signal()
	kq.reportQueueLength()
}

// SetLengthGauge sets the gauge for tracking queue length
func (kq *KeyedAppQueue) SetLengthGauge(g gaugeSetter) {
	kq.mu.Lock()
	defer kq.mu.Unlock()
	kq.lengthGauge = g
	kq.reportQueueLength()
}

func (kq *KeyedAppQueue) reportQueueLength() {
	if kq.lengthGauge != nil {
		kq.lengthGauge.Set(float64(kq.items.Len() + len(kq.pending)))
	}
}

// DequeueWait blocks until an item is available
func (kq *KeyedAppQueue) DequeueWait(ctx context.Context) (*AppQueueItem, bool) {
	kq.mu.Lock()
	defer kq.mu.Unlock()

	// Wait until queue has items or context is done
	for kq.items.Len() == 0 {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, false
		default:
		}
		// Wait for signal (releases lock while waiting)
		kq.notEmpty.Wait()
	}

	elem := kq.items.Front()
	item := elem.Value.(AppQueueItem)
	kq.items.Remove(elem)
	delete(kq.keyMap, item.Key)
	// put into processing set, so the same app won't
	// be processed by different gouroutines in parallel
	kq.processing[item.Key] = struct{}{}
	kq.reportQueueLength()
	return &item, true
}

// Done removes the key from the processing set. If any updates for the app
// were received while processing the app, the last one is requeued.
func (kq *KeyedAppQueue) Done(key string) {
	kq.mu.Lock()
	defer kq.mu.Unlock()

	if _, processing := kq.processing[key]; !processing {
		log.Errorf("Done called for app %q which is not being processed. this cannot happen!", key)
		return
	}
	delete(kq.processing, key)

	app, ok := kq.pending[key]
	if !ok {
		return
	}
	delete(kq.pending, key)
	kq.enqueueLocked(app)
}

type WorkerPool struct {
	queue   *KeyedAppQueue
	workers int
	jobChan chan *AppQueueItem
	ctx     context.Context
	cancel  context.CancelFunc
}

func (wp *WorkerPool) dequeueLoop() {
	for {
		select {
		case <-wp.ctx.Done():
			close(wp.jobChan)
			return
		default:
		}

		item, ok := wp.queue.DequeueWait(wp.ctx)
		if !ok {
			continue
		}

		select {
		case wp.jobChan <- item:
		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) Stop() {
	wp.cancel()
}

func NewWorkerPool(q *KeyedAppQueue, numWorkers int, f func(ctx context.Context, a appv1.Application)) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	wp := &WorkerPool{
		queue:   q,
		workers: numWorkers,
		jobChan: make(chan *AppQueueItem, numWorkers*2),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start workers
	for i := 0; i < numWorkers; i++ {
		go func() {
			for job := range wp.jobChan {
				log.Debugf("Worker %d: handling app  %q", i, job.Key)
				f(ctx, job.Value)
				wp.queue.Done(job.Key)
			}
		}()
	}
	go wp.dequeueLoop()
	return wp
}
