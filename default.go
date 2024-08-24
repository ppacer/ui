package ui

import (
	"fmt"
	"log"
	"net/http"
)

// DefaultStarted starts HTTP server which serves ppacer UI in default
// configuration. This function is meant to reduce boilerplate for simple
// examples and tests. When there is an error on starting UI server this
// function panics.
func DefaultStarted(schedulerPort, uiPort int) {
	schedulerUrl := fmt.Sprintf("http://localhost:%d", schedulerPort)
	uiDefault := NewUI(schedulerUrl, defaultLogger(), nil)
	portStr := fmt.Sprintf(":%d", uiPort)
	fmt.Println("Starting ppacer UI on ", portStr)
	err := http.ListenAndServe(portStr, uiDefault.Server())
	if err != nil {
		log.Panicf("Cannot start ppacer UI server: %s", err.Error())
	}
}

// DefaultStartedMocks starts HTTP server which serves ppacer UI in default
// configuration. Similarly to DefaultStarted, but instead of communicating
// with actual ppacer Scheduler it would used mocked data within the UI server.
// This function is primarily for local development convenience. When there is
// an error on starting UI server this function panics.
func DefaultStartedMocks(uiPort int) {
	uiDefault := NewUIWithMocks(defaultLogger(), nil)
	portStr := fmt.Sprintf(":%d", uiPort)
	fmt.Println("Starting ppacer UI with mocked data on ", portStr)
	err := http.ListenAndServe(portStr, uiDefault.Server())
	if err != nil {
		log.Panicf("Cannot start ppacer UI server: %s", err.Error())
	}
}
