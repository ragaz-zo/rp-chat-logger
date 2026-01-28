package main

import (
	"net/http"
	"net/url"
)

// parseMessage extracts the sender and message query parameters from
// an incoming HTTP request to the /message endpoint.
func parseMessage(r *http.Request) (string, string) {
	if r.URL.Path != "/message" {
		return "", ""
	}

	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return "", ""
	}

	sender := values.Get("sender")
	message := values.Get("message")

	if sender == "" || message == "" {
		return "", ""
	}

	return sender, message
}
