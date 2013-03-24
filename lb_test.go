// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lb

import (
	"container/heap"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sort"
	"testing"
)

type FakeHandler struct {
	msg      []byte
	requests []*http.Request
}

func (h *FakeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.requests = append(h.requests, r)
	w.Write(h.msg)
}

func TestBackendHandle(t *testing.T) {
	msg := "Hello!"
	handler := FakeHandler{msg: []byte(msg)}
	server := httptest.NewServer(&handler)
	defer server.Close()
	u, _ := url.Parse(server.URL)
	b := Backend{r: httputil.NewSingleHostReverseProxy(u)}
	path := "/hello"
	req, _ := http.NewRequest("GET", path, nil)
	recorder := httptest.NewRecorder()
	done := make(chan *Backend, 1)
	b.handle(recorder, req, done)
	b2 := <-done
	if &b != b2 {
		t.Errorf("Did not return the proper backend. Want %#v. Got %#v.", &b, b2)
	}
	body := recorder.Body.String()
	if body != msg {
		t.Errorf("Wrong response. Want %q. Got %q.", msg, body)
	}
	req = handler.requests[0]
	if req.URL.Path != path {
		t.Errorf("Wrong request path. Want %q. Got %q.", path, req.URL.Path)
	}
	if req.Method != "GET" {
		t.Errorf("Wrong request method. Want %q. Got %q.", "GET", req.Method)
	}
}

func TestPoolLen(t *testing.T) {
	b := []*Backend{
		{load: 1},
		{load: 2},
		{load: 3},
	}
	p := Pool(b)
	if p.Len() != len(b) {
		t.Errorf("Pool.Len: Want %d. Got %d.", len(b), p.Len())
	}
}

func TestPoolLess(t *testing.T) {
	b := []*Backend{
		{load: 2},
		{load: 1},
		{load: 3},
	}
	p := Pool(b)
	tests := []struct {
		i, j int
		less bool
	}{
		{0, 1, false},
		{1, 0, true},
		{1, 2, true},
		{0, 2, true},
		{2, 1, false},
		{0, 0, false},
		{2, 0, false},
	}
	for _, tt := range tests {
		got := p.Less(tt.i, tt.j)
		if got != tt.less {
			t.Errorf("Pool.Less(%d, %d). Want %v. Got %v.", tt.i, tt.j, tt.less, got)
		}
	}
}

func TestPoolSwap(t *testing.T) {
	b := []*Backend{
		{load: 2},
		{load: 1},
		{load: 3},
	}
	p := Pool(b)
	tests := []struct {
		i, j int
		less bool
	}{
		{0, 1, true},
		{0, 1, false},
		{1, 2, false},
		{1, 2, true},
	}
	for _, tt := range tests {
		p.Swap(tt.i, tt.j)
		got := p.Less(tt.i, tt.j)
		if got != tt.less {
			t.Errorf("Pool.Less(%d, %d) after Pool.Swap(%d, %d). Want %v. Got %v.", tt.i, tt.j, tt.i, tt.j, tt.less, got)
		}
	}
}

func TestPoolPush(t *testing.T) {
	bs := make([]*Backend, 0, 1)
	p := Pool(bs)
	b := Backend{i: -1}
	p.Push(&b)
	if b.i != 0 {
		t.Errorf("p.Push() should set Backend.i. Want %d. Got %d.", 0, b.i)
	}
	if p.Len() != 1 {
		t.Errorf("p.Push() should store backend. Want %d. Got %d.", 1, p.Len())
	}
}

func TestPoolPop(t *testing.T) {
	bs := []*Backend{
		{load: 2, i: 0},
		{load: 1, i: 1},
		{load: 0, i: 2},
	}
	p := Pool(bs)
	b := p.Pop().(*Backend)
	if b.i != -1 {
		t.Errorf("p.Pop() did not unset i. Want %d. Got %d.", -1, b.i)
	}
	if b.load != 0 {
		t.Errorf("p.Pop() did not get the right element. Want %d. Got %d.", 0, b.load)
	}
	if p.Len() != 2 {
		t.Errorf("p.Pop() did not remove the element from the slice. Want %d. Got %d.", 2, p.Len())
	}
}

func TestPoolPQ(t *testing.T) {
	expected := []int{5, 6, 14, 1, 2, 0, 4}
	bs := make([]*Backend, 0, len(expected))
	p := Pool(bs)
	for i, e := range expected {
		heap.Push(&p, &Backend{i: i, load: e})
	}
	sort.Ints(expected)
	for _, e := range expected {
		b := heap.Pop(&p).(*Backend)
		if b.load != e {
			t.Errorf("Did not return the proper backend. Want %d. Got %d.", e, b.load)
		}
	}
}

func BenchmarkPoolPushAndPop(b *testing.B) {
	for i := 1; i < b.N; i++ {
		bs := make([]*Backend, 0, i / 2)
		p := Pool(bs)
		for j := 0; j < i / 2; j++ {
			heap.Push(&p, &Backend{i: j, load: i + j})
		}
		for j := 0; j < i / 2; j++ {
			heap.Pop(&p)
		}
	}
}

func TestNewLoadBalancer(t *testing.T) {
	server1 := httptest.NewServer(&FakeHandler{msg: []byte("Hello from server 1.")})
	defer server1.Close()
	server2 := httptest.NewServer(&FakeHandler{msg: []byte("Hello from server 2.")})
	defer server2.Close()
	lb, err := NewLoadBalancer(server1.URL, server2.URL)
	if err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("GET", "http://globo.com/something/wrong", nil)
	copy := *req
	u1, _ := url.Parse(server1.URL)
	lb.p[0].r.Director(&copy)
	if copy.URL.Host != u1.Host {
		t.Errorf("LoadBalancer did not use reverse proxy. Want request host to be %q. Got %q.", u1.Host, copy.URL.Host)
	}
	u2, _ := url.Parse(server2.URL)
	copy = *req
	lb.p[1].r.Director(&copy)
	if copy.URL.Host != u2.Host {
		t.Errorf("LoadBalancer did not use reverse proxy. Want request host to be %q. Got %q.", u2.Host, copy.URL.Host)
	}
}

func TestNewLoadBalancerInvalidURL(t *testing.T) {
	lb, err := NewLoadBalancer("http://%%%%.com")
	if err == nil {
		t.Error("Expected non-nil error, got <nil>.")
	}
	if lb != nil {
		t.Errorf("Want <nil>. Got %#v.", *lb)
	}
}

func TestLoadBalancerHandle(t *testing.T) {
	h1 := &FakeHandler{msg: []byte("Hello from server 1.")}
	h2 := &FakeHandler{msg: []byte("Hello from server 2.")}
	server1 := httptest.NewServer(h1)
	defer server1.Close()
	server2 := httptest.NewServer(h2)
	defer server2.Close()
	req, _ := http.NewRequest("GET", "http://globo.com/something/wrong", nil)
	recorder := httptest.NewRecorder()
	lb, err := NewLoadBalancer(server1.URL, server2.URL)
	if err != nil {
		t.Fatal(err)
	}
	lb.Handle(recorder, req)
	<-lb.done
	var h *FakeHandler
	if len(h1.requests) > 0 {
		h = h1
	} else {
		h = h2
	}
	body := recorder.Body.String()
	if body != string(h.msg) {
		t.Errorf("Wrong response. Want %q. Got %q.", h.msg, body)
	}
	req = h.requests[0]
	if req.URL.Path != "/something/wrong" {
		t.Errorf("Wrong request path. Want %q. Got %q.", "/something/wrong", req.URL.Path)
	}
	if req.Method != "GET" {
		t.Errorf("Wrong request method. Want %q. Got %q.", "GET", req.Method)
	}
}
