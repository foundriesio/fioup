package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/foundriesio/composeapp/pkg/compose"
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
	switch state {
	case "Initializing":
		if !u.InitializedAt.IsZero() {
			fmt.Printf("done at %s; ", u.InitializedAt.UTC().Format(time.TimeOnly))
		}
	}
}

func postStateHandler(state api.StateName, update *api.UpdateInfo) {
	switch state {
	case "Checking":
		fmt.Printf("%s %s from %d [%s] to %d [%s]\n",
			update.Mode, update.Type,
			update.FromTarget.Version, strings.Join(update.FromTarget.AppNames(), ","),
			update.ToTarget.Version, strings.Join(update.ToTarget.AppNames(), ","))
	case "Initializing":
		fmt.Printf("update size: %s, %d blobs; add: [%s], remove: [%s], sync: [%s], update: [%s]\n",
			compose.FormatBytesInt64(update.Size.Bytes), update.Size.Blobs,
			strings.Join(update.AppDiff.Add.Names(), ","),
			strings.Join(update.AppDiff.Remove.Names(), ","),
			strings.Join(update.AppDiff.Sync.Names(), ","),
			strings.Join(update.AppDiff.Update.Names(), ","))
	default:
		fmt.Printf("done\n")
	}
}
