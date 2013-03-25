// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lb

import (
	"container/heap"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
)

type backend struct {
	i    int
	load counter
	r    *httputil.ReverseProxy
}

func (b *backend) handle(w http.ResponseWriter, r *http.Request, done chan<- *backend) {
	b.r.ServeHTTP(w, r)
	done <- b
}

type Pool struct {
	backends []*backend
	mut      sync.Mutex
}

func (p *Pool) Len() int {
	p.mut.Lock()
	defer p.mut.Unlock()
	return len(p.backends)
}

func (p *Pool) Less(i, j int) bool {
	return p.backends[i].load.val() < p.backends[j].load.val()
}

func (p *Pool) Swap(i, j int) {
	p.backends[i], p.backends[j] = p.backends[j], p.backends[i]
}

func (p *Pool) Push(x interface{}) {
	b := x.(*backend)
	b.i = p.Len()
	p.mut.Lock()
	defer p.mut.Unlock()
	p.backends = p.backends[:b.i+1]
	p.backends[b.i] = b
}

func (p *Pool) Pop() interface{} {
	l := p.Len() - 1
	p.mut.Lock()
	b := p.backends[l]
	p.backends = p.backends[:l]
	p.mut.Unlock()
	b.i = -1
	return b
}

type LoadBalancer struct {
	p    Pool
	done chan *backend
}

func NewLoadBalancer(hosts ...string) (*LoadBalancer, error) {
	backends := make([]*backend, 0, len(hosts))
	p := Pool{backends: backends}
	lb := LoadBalancer{
		p:    p,
		done: make(chan *backend, len(hosts)),
	}
	for _, h := range hosts {
		u, err := url.Parse(h)
		if err != nil {
			return nil, err
		}
		heap.Push(&lb.p, &backend{r: httputil.NewSingleHostReverseProxy(u)})
	}
	go lb.handleFinishes()
	return &lb, nil
}

func (l *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := heap.Pop(&l.p).(*backend)
	go b.handle(w, r, l.done)
	b.load.increment()
	heap.Push(&l.p, b)
}

func (l *LoadBalancer) requestFinished(b *backend) {
	heap.Remove(&l.p, b.i)
	b.load.decrement()
	heap.Push(&l.p, b)
}

func (l *LoadBalancer) handleFinishes() {
	for b := range l.done {
		go l.requestFinished(b)
	}
}
