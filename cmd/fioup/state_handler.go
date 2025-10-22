package main

import (
	"fmt"

	"github.com/foundriesio/fioup/pkg/api"
)

var (
	updateHandlers = []api.UpdateOpt{
		api.WithPreStateHandler(preStateHandler),
		api.WithPostStateHandler(postStateHandler),
	}
)

func preStateHandler(state api.StateName, ctx interface{}) {
	fmt.Printf("%s ... ", state)
}

func postStateHandler(state api.StateName, ctx interface{}) {
	fmt.Printf("done\n")
}
