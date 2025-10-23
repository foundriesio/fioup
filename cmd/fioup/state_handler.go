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
	case "Installing", "Starting":
		fmt.Println()
	}
}

func postStateHandler(state api.StateName, u *api.UpdateInfo) {
	switch state {
	case "Checking":
		fmt.Printf("%s %s from %d [%s] to %d [%s]\n",
			u.Mode, u.Type,
			u.FromTarget.Version, strings.Join(u.FromTarget.AppNames(), ","),
			u.ToTarget.Version, strings.Join(u.ToTarget.AppNames(), ","))
	case "Initializing":
		fmt.Printf("u size: %s, %d blobs; add: [%s], remove: [%s], sync: [%s], u: [%s]\n",
			compose.FormatBytesInt64(u.Size.Bytes), u.Size.Blobs,
			strings.Join(u.AppDiff.Add.Names(), ","),
			strings.Join(u.AppDiff.Remove.Names(), ","),
			strings.Join(u.AppDiff.Sync.Names(), ","),
			strings.Join(u.AppDiff.Update.Names(), ","))
	case "Fetching":
		if u.Size.Bytes == 0 {
			fmt.Print("done\n")
		} else {
			fmt.Println()
		}
	case "Installing", "Starting":
		fmt.Print("      Done\n")
	default:
		fmt.Printf("done\n")
	}
}

func appStartHandler(app compose.App, status compose.AppStartStatus, any interface{}) {
	switch status {
	case compose.AppStartStatusStarting:
		fmt.Printf("\tstarting %s --> %s ... ", app.Name(), app.Ref().String())
	case compose.AppStartStatusStarted:
		fmt.Println("done")
	case compose.AppStartStatusFailed:
		fmt.Println("failed")
	}
}
