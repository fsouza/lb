// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package lb contains an HTTP load balancer implementation.
//
// It was developed for teaching purposes, prepared for a Go workshop at
// Globo.com.
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

type pool struct {
	backends []*backend
	mut      sync.Mutex
}

func (p *pool) Len() int {
	p.mut.Lock()
	defer p.mut.Unlock()
	return len(p.backends)
}

func (p *pool) Less(i, j int) bool {
	return p.backends[i].load.val() < p.backends[j].load.val()
}

func (p *pool) Swap(i, j int) {
	p.backends[i], p.backends[j] = p.backends[j], p.backends[i]
}

func (p *pool) Push(x interface{}) {
	b := x.(*backend)
	b.i = p.Len()
	p.mut.Lock()
	defer p.mut.Unlock()
	p.backends = p.backends[:b.i+1]
	p.backends[b.i] = b
}

func (p *pool) Pop() interface{} {
	l := p.Len() - 1
	p.mut.Lock()
	b := p.backends[l]
	p.backends = p.backends[:l]
	p.mut.Unlock()
	b.i = -1
	return b
}

// LoadBalancer represents an HTTP load balancer. It implements a fair
// scheduling model, dispatching new requests to the backend with least load.
//
// It implements http.Handler, so you can map a resource in a Go HTTP server to
// a load balancer.
type LoadBalancer struct {
	p    pool
	done chan *backend
}

// NewLoadBalancer returns a new instance of a LoadBalancer. It receives one or
// more hosts (represented by URLs), and balances the load between them.
func NewLoadBalancer(hosts ...string) (*LoadBalancer, error) {
	backends := make([]*backend, 0, len(hosts))
	p := pool{backends: backends}
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
	b.load.increment()
	heap.Push(&l.p, b)
	b.handle(w, r, l.done)
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
