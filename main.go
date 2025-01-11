package promise

import (
	"sync"
)

type State string

const (
	StatePending  State = "PENDING"
	StateResolved State = "RESOLVED"
	StateRejected State = "REJECTED"
)

type result struct {
	state State
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
	result result
}

func (p *Promise) poll() result {
	resolver := func(result ...any) result {
		p.result.state = StateResolved
		if len(result) > 0 {
			p.result.value = result[0]
		}

		return p.result
	}
	rejector := func(err ...error) result {
		p.result.state = StateRejected
		if len(err) > 0 {
			p.result.err = err[0]
		}

		return p.result
	}

	go p.fn(resolver, rejector)

	return p.result
}

func (p *Promise) Then(resolve ThenFn) *Promise {
	return New(func(_resolve Resolver, reject Rejector) {
		value := resolve(p.result.value)
		_resolve(value)
	})
}

func (p *Promise) Catch(reject CatchFn) *Promise {
	return New(func(resolve Resolver, _reject Rejector) {
		err := reject(p.result.err)
		_reject(err)
	})
}

type scheduler struct {
	promises map[*Promise]bool
}

func (r *scheduler) add(promise *Promise) {
	r.promises[promise] = true
}

func (r *scheduler) done() bool {
	return len(r.promises) == 0
}

func (r *scheduler) remove(promise *Promise) {
	delete(r.promises, promise)
}

func (r *scheduler) run() {
	for !r.done() {
		for promise := range r.promises {
			result := promise.poll()
			if result.state == StatePending {
				continue
			}
			r.remove(promise)
		}
	}
}

var runner = &scheduler{
	promises: make(map[*Promise]bool),
}

func New(fn PromiseFn) *Promise {
	promise := &Promise{
		fn: fn,
		result: result{
			state: StatePending,
		},
	}
	runner.add(promise)
	return promise
}

func Await(promise *Promise) (any, error) {
	result := promise.poll()

	if result.state == StateResolved {
		return result.value, nil
	}

	if result.state == StateRejected {
		return nil, result.err
	}

	runner.remove(promise)
	return nil, nil
}

var once sync.Once

func init() {
	once.Do(func() {
		go runner.run()
	})
}
