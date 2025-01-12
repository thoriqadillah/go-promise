package promise

import (
	"sync"
)

type state string

const (
	statePending  state = "PENDING"
	stateResolved state = "RESOLVED"
	stateRejected state = "REJECTED"
)

type result struct {
	state state
	value any
	err   error
}

type channel struct {
	listener []func(data ...any)
	lock     sync.Mutex
}

func (e *channel) subscribe(fn func(data ...any)) {
	e.lock.Lock()
	defer e.lock.Unlock()
	e.listener = append(e.listener, fn)
}

func (e *channel) send(data ...any) {
	for _, fn := range e.listener {
		fn(data...)
	}
}

type Resolver = func(result ...any)
type Rejector = func(err ...error)

type PromiseFn = func(resolve Resolver, reject Rejector)

type ThenFn = func(result any) any
type CatchFn = func(err error) error

type Promise struct {
	fn    PromiseFn
	event *channel
}

func (p *Promise) poll() {
	resolver := func(res ...any) {
		var r result
		r.state = stateResolved
		if len(res) > 0 {
			r.value = res[0]
		}

		p.event.send(r)
	}

	rejector := func(err ...error) {
		var r result
		r.state = stateRejected
		if len(err) > 0 {
			r.err = err[0]
		}

		p.event.send(r)
	}

	go p.fn(resolver, rejector)
}

func (p *Promise) Then(then ThenFn) *Promise {
	return New(func(resolve Resolver, reject Rejector) {
		p.event.subscribe(func(data ...any) {
			result := data[0].(result)
			value := then(result.value)
			resolve(value)
		})
	})
}

func (p *Promise) Catch(catch CatchFn) *Promise {
	return New(func(resolve Resolver, reject Rejector) {
		p.event.subscribe(func(data ...any) {
			result := data[0].(result)
			err := catch(result.err)
			reject(err)
		})
	})
}

type scheduler struct {
	promises chan *Promise
}

func (r *scheduler) add(promise *Promise) {
	r.promises <- promise
}

func (r *scheduler) run() {
	for promise := range r.promises {
		promise.poll()
	}
}

var runner = &scheduler{
	promises: make(chan *Promise),
}

func New(fn PromiseFn) *Promise {
	promise := &Promise{
		fn: fn,
		event: &channel{
			listener: make([]func(data ...any), 0),
			lock:     sync.Mutex{},
		},
	}
	runner.add(promise)
	return promise
}

func Await(promise *Promise) (any, error) {
	var mu sync.Mutex
	mu.Lock()
	var r result
	r.state = statePending

	promise.event.subscribe(func(data ...any) {
		r = data[0].(result)
		mu.Unlock()
	})

	mu.Lock()
	defer mu.Unlock()

	if r.state == stateResolved {
		return r.value, nil
	}

	if r.state == stateRejected {
		return nil, r.err
	}

	return nil, nil
}

func All(promises ...*Promise) *Promise {
	return New(func(resolve Resolver, reject Rejector) {
		results := make([]any, len(promises))
		for _, promise := range promises {
			promise.Then(func(result any) any {
				results = append(results, result)
				return result
			}).Catch(func(err error) error {
				reject(err)
				return err
			})
		}

		resolve(results)
	})
}

var once sync.Once

func init() {
	once.Do(func() {
		go runner.run()
	})
}
