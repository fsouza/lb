// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lb

import (
	"sync/atomic"
)

type counter struct {
	v int64
}

func newCounter(initial int64) *counter {
	return &counter{v: initial}
}

func (c *counter) val() int64 {
	return atomic.LoadInt64(&c.v)
}

func (c *counter) increment() {
	old := atomic.LoadInt64(&c.v)
	swapped := atomic.CompareAndSwapInt64(&c.v, old, old+1)
	for !swapped {
		old = atomic.LoadInt64(&c.v)
		swapped = atomic.CompareAndSwapInt64(&c.v, old, old+1)
	}
}

func (c *counter) decrement() {
	old := atomic.LoadInt64(&c.v)
	swapped := atomic.CompareAndSwapInt64(&c.v, old, old-1)
	for !swapped {
		old = atomic.LoadInt64(&c.v)
		swapped = atomic.CompareAndSwapInt64(&c.v, old, old-1)
	}
}
