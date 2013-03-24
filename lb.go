// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lb

import (
	"container/heap"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type Backend struct {
	i    int
	load counter
	r    *httputil.ReverseProxy
}

func (b *Backend) handle(w http.ResponseWriter, r *http.Request, done chan<- *Backend) {
	b.r.ServeHTTP(w, r)
	done <- b
}

type Pool []*Backend

func (p *Pool) Len() int {
	return len(*p)
}

func (p *Pool) Less(i, j int) bool {
	return (*p)[i].load.val() < (*p)[j].load.val()
}

func (p *Pool) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}

func (p *Pool) Push(x interface{}) {
	b := x.(*Backend)
	b.i = p.Len()
	*p = (*p)[:b.i+1]
	(*p)[b.i] = b
}

func (p *Pool) Pop() interface{} {
	b := (*p)[p.Len()-1]
	b.i = -1
	(*p) = (*p)[:p.Len()-1]
	return b
}

type LoadBalancer struct {
	p    Pool
	done chan *Backend
}

func NewLoadBalancer(hosts ...string) (*LoadBalancer, error) {
	backends := make([]*Backend, 0, len(hosts))
	p := Pool(backends)
	lb := LoadBalancer{
		p:    p,
		done: make(chan *Backend, len(hosts)),
	}
	for _, h := range hosts {
		u, err := url.Parse(h)
		if err != nil {
			return nil, err
		}
		heap.Push(&lb.p, &Backend{r: httputil.NewSingleHostReverseProxy(u)})
	}
	go lb.handleFinishes()
	return &lb, nil
}

func (l *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	b := heap.Pop(&l.p).(*Backend)
	go b.handle(w, r, l.done)
	b.load.increment()
	heap.Push(&l.p, b)
}

func (l *LoadBalancer) requestFinished(b *Backend) {
	heap.Remove(&l.p, b.i)
	b.load.decrement()
	heap.Push(&l.p, b)
}

func (l *LoadBalancer) handleFinishes() {
	for b := range l.done {
		go l.requestFinished(b)
	}
}
