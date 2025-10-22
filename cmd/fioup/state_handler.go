package main

import (
	"fmt"
	"strings"

	"github.com/foundriesio/fioup/pkg/api"
)

var (
	updateHandlers = []api.UpdateOpt{
		api.WithPreStateHandler(preStateHandler),
		api.WithPostStateHandler(postStateHandler),
	}
)

func preStateHandler(state api.StateName, u *api.UpdateInfo) {
	fmt.Printf("[%d/%d] %s ... ", u.CurrentStateNum, u.TotalStates, state)
}

func postStateHandler(state api.StateName, update *api.UpdateInfo) {
	switch state {
	case "Checking":
		fmt.Printf("%s %s from %d [%s] to %d [%s]\n",
			update.Mode, update.Type,
			update.FromTarget.Version, strings.Join(update.FromTarget.AppNames(), ","),
			update.ToTarget.Version, strings.Join(update.ToTarget.AppNames(), ","))
	default:
		fmt.Printf("done\n")
	}
}
