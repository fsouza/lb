// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lb

import (
	"net/http"
	"net/http/httputil"
)

type Backend struct {
	R *httputil.ReverseProxy
}

func (b *Backend) handle(w http.ResponseWriter, r *http.Request, done chan<- *Backend) {
	b.R.ServeHTTP(w, r)
	done <- b
}
