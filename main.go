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

type future interface {
	poll() result
}

type Resolver = func(result ...any) result
type Rejector = func(err ...error) result

type PromiseFn = func(resolve Resolver, reject Rejector)

type ThenFn = func(result any) any
type CatchFn = func(err error) error

type Promise struct {
	fn     PromiseFn
	result chan result
}

func (p *Promise) poll() {
	resolver := func(res ...any) result {
		var r result
		r.state = stateResolved
		if len(res) > 0 {
			r.value = res[0]
		}

		p.result <- r
		return r
	}

	rejector := func(err ...error) result {
		var r result
		r.state = stateRejected
		if len(err) > 0 {
			r.err = err[0]
		}

		p.result <- r
		return r
	}

	go p.fn(resolver, rejector)
}

func (p *Promise) Then(resolve ThenFn) *Promise {
	return New(func(_resolve Resolver, reject Rejector) {
		result := <-p.result
		value := resolve(result.value)
		_resolve(value)
	})
}

func (p *Promise) Catch(reject CatchFn) *Promise {
	return New(func(resolve Resolver, _reject Rejector) {
		result := <-p.result
		err := reject(result.err)
		_reject(err)
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
		fn:     fn,
		result: make(chan result),
	}
	runner.add(promise)
	return promise
}

func Await(promise *Promise) (any, error) {
	result := <-promise.result

	if result.state == stateResolved {
		return result.value, nil
	}

	if result.state == stateRejected {
		return nil, result.err
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
