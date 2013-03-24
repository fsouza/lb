// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lb

import (
	"net/http"
	"net/http/httputil"
)

type Backend struct {
	load int
	R    *httputil.ReverseProxy
}

func (b *Backend) handle(w http.ResponseWriter, r *http.Request, done chan<- *Backend) {
	b.R.ServeHTTP(w, r)
	done <- b
}

type Pool []*Backend

func (p *Pool) Len() int {
	return len(*p)
}

func (p *Pool) Less(i, j int) bool {
	return (*p)[i].load < (*p)[j].load
}

func (p *Pool) Swap(i, j int) {
	(*p)[i], (*p)[j] = (*p)[j], (*p)[i]
}
