package handlers_test

import "net/http/cookiejar"

func newJar() (*cookiejar.Jar, error) { return cookiejar.New(nil) }
