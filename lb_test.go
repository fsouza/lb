// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lb

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
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
	b := Backend{R: httputil.NewSingleHostReverseProxy(u)}
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
