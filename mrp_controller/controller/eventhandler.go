package controller

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/watch"

	appv1 "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
)

type subscriber struct {
	eventHandler func(event *appv1.ApplicationWatchEvent) error
	filters      []func(*appv1.ApplicationWatchEvent) bool
}

func (s *subscriber) matches(event *appv1.ApplicationWatchEvent) bool {
	for i := range s.filters {
		if !s.filters[i](event) {
			return false
		}
	}
	return true
}

// AppEventHandler is an interface for broadcasting application informer watch events to multiple subscribers.
type AppEventHandler interface {
	Subscribe(handlerFunc func(event *appv1.ApplicationWatchEvent) error, filters ...func(event *appv1.ApplicationWatchEvent) bool) func()
	OnAdd(any, bool)
	OnUpdate(any, any)
	OnDelete(any)
}

type appEventHandler struct {
	lock        sync.Mutex
	subscribers []*subscriber
}

func NewAppEventHandler() AppEventHandler {
	return &appEventHandler{}
}

func (b *appEventHandler) notify(event *appv1.ApplicationWatchEvent) {
	// Make a local copy of b.subscribers, then send channel events outside the lock,
	// to avoid data race on b.subscribers changes
	subscribers := []*subscriber{}
	b.lock.Lock()
	subscribers = append(subscribers, b.subscribers...)
	b.lock.Unlock()

	for _, s := range subscribers {
		if s.matches(event) {
			err := s.eventHandler(event)
			if err != nil {
				log.Errorf("failing to handle event '%v' on %s/%s", event.Type,
					event.Application.Namespace, event.Application.Name)
			}
		}
	}
}

// Subscribe forward application informer watch events to the provided function
func (b *appEventHandler) Subscribe(handlerFunc func(event *appv1.ApplicationWatchEvent) error, filters ...func(event *appv1.ApplicationWatchEvent) bool) func() {
	b.lock.Lock()
	defer b.lock.Unlock()
	subscriber := &subscriber{handlerFunc, filters}
	b.subscribers = append(b.subscribers, subscriber)
	return func() {
		b.lock.Lock()
		defer b.lock.Unlock()
		for i := range b.subscribers {
			if b.subscribers[i] == subscriber {
				b.subscribers = append(b.subscribers[:i], b.subscribers[i+1:]...)
				break
			}
		}
	}
}

func (b *appEventHandler) OnAdd(obj any, _ bool) {
	if app, ok := obj.(*appv1.Application); ok {
		b.notify(&appv1.ApplicationWatchEvent{Application: *app, Type: watch.Added})
	}
}

func (b *appEventHandler) OnUpdate(_, newObj any) {
	if app, ok := newObj.(*appv1.Application); ok {
		b.notify(&appv1.ApplicationWatchEvent{Application: *app, Type: watch.Modified})
	}
}

func (b *appEventHandler) OnDelete(obj any) {
	if app, ok := obj.(*appv1.Application); ok {
		b.notify(&appv1.ApplicationWatchEvent{Application: *app, Type: watch.Deleted})
	}
}
