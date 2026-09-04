package main

import (
	"net/http"
	"testing"
)

func TestServeReportsUnexpectedServerError(t *testing.T) {
	serverErr := make(chan error, 1)
	serve(&http.Server{Addr: "invalid-address"}, serverErr)

	if err := <-serverErr; err == nil {
		t.Fatal("serve() did not report server error")
	}
}
