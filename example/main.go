package main

import (
	"fmt"
	"log"
	"promise"
	"sync"
	"time"
)

func wait(ms int) *promise.Promise {
	return promise.New(func(resolve promise.Resolver, reject promise.Rejector) {
		time.Sleep(time.Duration(ms) * time.Millisecond)
		resolve(fmt.Sprintf("waited for %d ms", ms))
	})
}

func waitCall(n int, ms int) {
	wait(ms).Then(func(result any) any {
		fmt.Printf("%d has waited for %d ms\n", n, ms)
		return nil
	})
}

func main() {
	var wg sync.WaitGroup
	wg.Add(10)

	go func() {
		i := 1
		ticker := time.NewTicker(1 * time.Second)
		for {
			select {
			case <-ticker.C:
				log.Printf("%d s\n", i)
				i++
				wg.Done()
			}
		}
	}()

	res, _ := promise.Await(wait(2000))
	log.Println(res)

	waitCall(1, 1000)
	waitCall(3, 2000)
	waitCall(2, 3000)
	waitCall(4, 2000)

	wg.Wait()
}
