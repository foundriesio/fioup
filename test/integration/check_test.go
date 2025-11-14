package e2e_tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"

	f "github.com/foundriesio/composeapp/test/fixtures"
	"github.com/foundriesio/fioconfig/sotatoml"
	"github.com/foundriesio/fioconfig/transport"
	"github.com/foundriesio/fioup/internal/events"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/client"
	cfg "github.com/foundriesio/fioup/pkg/config"
)

var (
	config *cfg.Config
)

type mockHttpOperations struct{}

func (mockHttpOperations) HttpGet(client *http.Client, url string, headers map[string]string) (*transport.HttpRes, error) {
	filePath := strings.Replace(url, config.GetServerBaseURL().String(), "../artifacts", 1)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to read mock file %s: %w", filePath, err)
	}

	res := transport.HttpRes{
		StatusCode: 200,
		Body:       data,
		Header:     http.Header{},
	}

	fmt.Print("HTTPGET! " + url + " -> " + filePath + "\n")
	return &res, nil
}

var postedEvents []events.DgUpdateEvent

func (mockHttpOperations) HttpDo(client *http.Client, method, url string, headers map[string]string, data any) (*transport.HttpRes, error) {
	filePath := strings.Replace(url, config.GetServerBaseURL().String(), "../output_artifacts", 1)
	// os.WriteFile(filePath, data.([]byte), 0o600)
	fmt.Print("HTTPDO! " + method + " " + url + " -> " + filePath + "\n")
	if method == http.MethodPost {
		if strings.HasSuffix(url, "/events") {
			postedEvents = append(postedEvents, data.([]events.DgUpdateEvent)...)
		}
	}

	res := transport.HttpRes{
		StatusCode: 200,
		Header:     http.Header{},
	}

	return &res, nil
}

func checkEvents(t *testing.T, target *Target) {
	for _, ev := range postedEvents {
		fmt.Printf("Event: %+v\n", ev)
		if ev.Event.Success != nil {
			fmt.Printf("success: %t\n", *ev.Event.Success)
		}
		evtVersion, err := strconv.Atoi(ev.Event.Version)
		if err != nil {
			t.Fatalf("Event version is not an integer: %v", err)
		}

		if evtVersion != target.Version {
			t.Fatalf("Event version does not match target version")
		}
	}
	postedEvents = []events.DgUpdateEvent{}
}

func TestCheck(t *testing.T) {
	client.DefaultHttpOperations = mockHttpOperations{}
	var configPaths []string = sotatoml.DEF_CONFIG_ORDER
	cfg, err := cfg.NewConfig(configPaths)
	config = cfg
	if err != nil {
		t.Fatalf("Unable to create config: %v", err)
	}

	target1 := genNewTarget(t, 100, 2, 50)
	target2 := genNewTarget(t, 101, 3, 60)
	target3 := genNewTarget(t, 102, 1, 70)

	saveTargetsJson([]*Target{target1})
	testUpdateTo(t, target1)

	api.Cancel(context.Background(), config)

	saveTargetsJson([]*Target{target1, target2})
	testUpdateTo(t, target2)

	saveTargetsJson([]*Target{target1, target2, target3})
	testUpdateTo(t, target3)
}

func testUpdateTo(t *testing.T, target *Target) {
	targets, currentStatus, err := api.Check(context.Background(), config, api.WithTUF(false))
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}
	fmt.Printf("Targets: %v\n", targets)
	fmt.Printf("Current Status: %v\n", currentStatus)

	err = api.Fetch(context.Background(), config, -1, api.WithTUF(false))
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	fmt.Println("Fetch events:")
	checkEvents(t, target)

	err = api.Install(context.Background(), config)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	fmt.Println("Install events")
	checkEvents(t, target)

	err = api.Start(context.Background(), config)
	if err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	fmt.Println("Start events:")
	checkEvents(t, target)
}

type Target struct {
	Version int
	Apps    []*f.App
}

func saveTargetsJson(targets []*Target) {
	targetsStr := []string{}
	for _, target := range targets {
		appsStr := ""
		for _, app := range target.Apps {
			appsStr += fmt.Sprintf(`"%s": {"uri": "%s"},`, app.Name, app.PublishedUri)
		}
		appsStr = strings.TrimSuffix(appsStr, ",")

		targetStr := fmt.Sprintf(`
      "intel-corei7-64-lmp-%d": {
        "custom": {
          "arch": "x86_64",
          "containers-sha": "5c0d79ba139344e856f844299bcda57880e1ec46",
          "createdAt": "2025-05-28T15:31:29Z",
          "docker_compose_apps": {%s},
          "hardwareIds": [
            "intel-corei7-64"
          ],
          "image-file": "lmp-base-console-image-intel-corei7-64.wic.gz",
          "lmp-manifest-sha": "e679f9c4f52ea6c94c66359b1b82b8feae48c7ca",
          "lmp-ver": "4.0.20-2-94.1",
          "meta-subscriber-overrides-sha": "117235537fcc10c64ce68164c86f433406b42f03",
          "name": "intel-corei7-64-lmp",
          "origUriApps": "https://ci.foundries.io/projects/factory/lmp/builds/422",
          "tags": [
            "e2e-test-temp-1"
          ],
          "targetFormat": "OSTREE",
          "updatedAt": "2025-05-28T15:31:29Z",
          "version": "%d"
        },
        "hashes": {
          "sha256": "1cb4e739e9fc8d823f8bc6bd0a4c4da144d54f0bee9d49165026ae3290e4e282"
        },
        "length": 0
      }`, target.Version, appsStr, target.Version)
		targetsStr = append(targetsStr, targetStr)
	}
	targetsJson := strings.Join(targetsStr, ",")
	fullJson := fmt.Sprintf(`
{
  "signatures": [
    {
      "keyid": "c89acf389bc9b058fbbd232a278919c9809384f38ced2c894a5ea80170b50919",
      "method": "ed25519",
      "sig": "w2hzJ/0sLbN4XVYUZrYNrQjQEBfivyPAiw41+0XfTL2Jc+aZ7iTBr4LmxREpLTi3xiys71QsJfi/BantpEJ9Aw=="
    }
  ],
  "signed": {
    "_type": "Targets",
    "expires": "2025-12-07T06:10:56Z",
    "targets": {
	  %s
    },
    "version": 352
  }
}
`, targetsJson)

	os.WriteFile("../artifacts/repo/targets.json", []byte(fullJson), 0o600)
}

func genNewTarget(t *testing.T, version int, numberOfApps int, portOffset int) *Target {
	appComposeDef := `
services:
  srvs-01:
    image: registry:5000/factory/runner-image:v0.1
    command: sh -c "while true; do sleep 60; done"
    ports:
    - %d:80
  busybox:
    image: ghcr.io/foundriesio/busybox:1.36
    command: sh -c "while true; do sleep 60; done"
`
	apps := []*f.App{}

	for i := 1; i <= numberOfApps; i++ {
		port := 8080 + i + portOffset
		app := f.NewApp(t, fmt.Sprintf(appComposeDef, port))
		app.Publish(t)

		app.Pull(t)
		defer app.Remove(t)
		// app.CheckFetched(t)
		apps = append(apps, app)
	}
	return &Target{
		Version: version,
		Apps:    apps,
	}
}
