package main

import (
	"fmt"
	"os"
	"os/signal"
)

var interruptChannel chan os.Signal

var addHandlerChannel = make(chan func())

var interruptHandlersDone = make(chan struct{})

var signals = []os.Signal{
	os.Interrupt,
}

func mainInterruptHandler() {
	var interruptCallbacks []func()
	invokeCallbacks := func() {
		for i := range interruptCallbacks {
			idx := len(interruptCallbacks) - 1 - i
			interruptCallbacks[idx]()
		}
		close(interruptHandlersDone)
	}

	for {
		select {
		case sig := <-interruptChannel:
			fmt.Printf("Received signal (%s). Shutting down...", sig)
			invokeCallbacks()
			return
		case handler := <-addHandlerChannel:
			interruptCallbacks = append(interruptCallbacks, handler)
		}
	}
}

func addInterruptHandler(handler func()) {
	if interruptChannel == nil {
		interruptChannel = make(chan os.Signal, 1)
		signal.Notify(interruptChannel, signals...)
		go mainInterruptHandler()
	}

	addHandlerChannel <- handler
}
