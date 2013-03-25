// Copyright 2013 Francisco Souza. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package lb_test

import (
	"github.com/fsouza/lb"
	"log"
	"net/http"
)

// This example demonstrates the use of a LoadBalancer as an http.Handler.
func ExampleLoadBalancer() {
	balancer, err := lb.NewLoadBalancer("localhost:8080", "localhost:8081", "localhost:8082")
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", balancer)
	log.Fatal(http.ListenAndServe(":80", nil))
}
