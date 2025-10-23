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
	case "Fetching":
		if u.Size.Bytes == 0 {
			fmt.Printf("nothing to fetch; ")
		} else if !u.FetchedAt.IsZero() {
			fmt.Printf("done at %s; fetched %s, %d blobs",
				u.FetchedAt.UTC().Format(time.TimeOnly),
				compose.FormatBytesInt64(u.Size.Bytes), u.Size.Blobs)
		} else {
			// Print fetch progress in the next line
			fmt.Println()
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
	case "Fetching":
		if update.Size.Bytes == 0 {
			fmt.Print("done\n")
		} else {
			fmt.Println()
		}
	default:
		fmt.Printf("done\n")
	}
}
