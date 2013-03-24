// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build race

package lb

import (
	"container/heap"
	"sync"
	"testing"
	"time"
)

func TestPoolIsSafe(t *testing.T) {
	bs := make([]*Backend, 0, 200)
	p := Pool{backends: bs}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		for i := 0; i < 180; i++ {
			heap.Push(&p, &Backend{})
		}
		wg.Done()
	}()
	time.Sleep(1e6)
	go func() {
		for i := 0; i < 50; i++ {
			heap.Pop(&p)
		}
		wg.Done()
	}()
	wg.Wait()
}