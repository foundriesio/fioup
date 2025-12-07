package integration_tests

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	f "github.com/foundriesio/composeapp/test/fixtures"
	"github.com/foundriesio/fioup/pkg/api"
	"github.com/foundriesio/fioup/pkg/client"
	cfg "github.com/foundriesio/fioup/pkg/config"
	"github.com/foundriesio/fioup/pkg/status"
	"github.com/stretchr/testify/assert"
)

const pkey_pem = `
-----BEGIN PRIVATE KEY-----
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgfk24YU2ArBZ99NMX
wO4+BmzTKzjbEGQwiVSJhqUIq1ahRANCAAQ0ZJoEcRLvF2rx1oJbbJ+K9fVjZUR9
Kk7giZQYgD8hd3ZWU9vR81b7eRMq5w0Wy9bmTp9nEi0LYqonbx98WKu5
-----END PRIVATE KEY-----`

const client_pem = `
-----BEGIN CERTIFICATE-----
MIIBgjCCASmgAwIBAgIRAJjpxA3hJU0jqfFeQkV+bgcwCgYIKoZIzj0EAwIwGTEX
MBUGA1UEAwwOb3RhLWRldmljZXMtQ0EwHhcNMjAwNjE3MTg0MjA3WhcNNDAwNjEy
MTg0MjA3WjBBMRAwDgYDVQQLDAdkZWZhdWx0MS0wKwYDVQQDDCQ5OGU5YzQwZC1l
MTI1LTRkMjMtYTlmMS01ZTQyNDU3ZTZlMDcwWTATBgcqhkjOPQIBBggqhkjOPQMB
BwNCAAQ0ZJoEcRLvF2rx1oJbbJ+K9fVjZUR9Kk7giZQYgD8hd3ZWU9vR81b7eRMq
5w0Wy9bmTp9nEi0LYqonbx98WKu5oyowKDAOBgNVHQ8BAf8EBAMCB4AwFgYDVR0l
AQH/BAwwCgYIKwYBBQUHAwIwCgYIKoZIzj0EAwIDRwAwRAIgPD6QZGSr1svchGAW
Jz2r/9CP9uby6JEzSrq2B0zkBewCIEKwxI/9j44n2NB8fzMOKbxAwKkI1sNTQRoJ
LSzKq+SZ
-----END CERTIFICATE-----
`

const root_crt = `
-----BEGIN CERTIFICATE-----
MIIDmzCCAyCgAwIBAgISBpfC7mgqDlDYYl29qvORucRYMAoGCCqGSM49BAMDMDIx
CzAJBgNVBAYTAlVTMRYwFAYDVQQKEw1MZXQncyBFbmNyeXB0MQswCQYDVQQDEwJF
ODAeFw0yNTA5MjQwNjU5NTVaFw0yNTEyMjMwNjU5NTRaMBcxFTATBgNVBAMTDGZv
dW5kcmllcy5pbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABONrCMWVCI8xAoM4
7RM2BJjFbQ6jIJuDiAJFq9bnM15de4GJo+0u7+LEGoWNXt4zIs8YwXYMh+DE8YPC
AGDzRjyjggIvMIICKzAOBgNVHQ8BAf8EBAMCB4AwHQYDVR0lBBYwFAYIKwYBBQUH
AwEGCCsGAQUFBwMCMAwGA1UdEwEB/wQCMAAwHQYDVR0OBBYEFBVt9MPiTOMjD96y
HpDM2QZf5YvtMB8GA1UdIwQYMBaAFI8NE6L2Ln7RUGwzGDhdWY4jcpHKMDIGCCsG
AQUFBwEBBCYwJDAiBggrBgEFBQcwAoYWaHR0cDovL2U4LmkubGVuY3Iub3JnLzAs
BgNVHREEJTAjggxmb3VuZHJpZXMuaW+CE2ZzbGluay5mb3VuZHJpZXMuaW8wEwYD
VR0gBAwwCjAIBgZngQwBAgEwLQYDVR0fBCYwJDAioCCgHoYcaHR0cDovL2U4LmMu
bGVuY3Iub3JnLzg5LmNybDCCAQQGCisGAQQB1nkCBAIEgfUEgfIA8AB3ABLxTjS9
U3JMhAYZw48/ehP457Vih4icbTAFhOvlhiY6AAABmXq7FYUAAAQDAEgwRgIhAI3u
kvpeSAY6TD2yMCHNCz1ZPeKG9fREj5oQyZZ/LnY2AiEAhBXftdgfLx1/omZ36emH
jQxePY9iS5NFBcYToLd/zgcAdQDtPEvW6AbCpKIAV9vLJOI4Ad9RL+3EhsVwDyDd
tz4/4AAAAZl6uxWYAAAEAwBGMEQCIFgc7poynbFL1UEuRSGOZA1pUpF6J80o03nB
7hvVQ2d6AiBzs8UqU2HdM4TRm/LPAqJVYsAJ69aNzknZKBIrAWcKeDAKBggqhkjO
PQQDAwNpADBmAjEA4RcMpsI0cKI3TyO7nGRqBl7cqWE0VOwghxgi+kjzmp8y3WRQ
WNCX+QvTXbOk6qA7AjEAjFORx0MrHYFxWE4h4khlwZRzXqTydwTo/KVYYo5YYs9R
AG/LlZuut3Px7oGWZ+wx
-----END CERTIFICATE-----
`

func createMockConfig(t *testing.T, tempDir string) *cfg.Config {
	if tempDir == "" {
		t.Fatal("tempDir not set")
	}
	if err := os.WriteFile(filepath.Join(tempDir, "root.crt"), []byte(root_crt), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "pkey.pem"), []byte(pkey_pem), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tempDir, "client.pem"), []byte(client_pem), 0644); err != nil {
		t.Fatal(err)
	}
	sota := fmt.Sprintf(`
[tls]
server = "https://example.com:8443"
ca_source = "file"
pkey_source = "file"
cert_source = "file"

[import]
tls_cacert_path = "%s/root.crt"
tls_pkey_path = "%s/pkey.pem"
tls_clientcert_path = "%s/client.pem"

[pacman]
tags = "main"
reset_apps_root = "%s/reset-apps"
compose_apps_root = "%s/compose-apps"

[provision]
primary_ecu_hardware_id = "intel-corei7-64"

[storage]
path = "%s"
	`, tempDir, tempDir, tempDir, tempDir, tempDir, tempDir)
	if err := os.WriteFile(filepath.Join(tempDir, "sota.toml"), []byte(sota), 0644); err != nil {
		t.Fatal(err)
	}

	config, err := cfg.NewConfig([]string{tempDir})
	if err != nil {
		t.Fatalf("Unable to create config: %v", err)
	}
	return config
}

func (it *integrationTest) refreshConfig() {
	config, err := cfg.NewConfig([]string{it.tempDir})
	if err != nil {
		it.t.Fatalf("Unable to create config: %v", err)
	}
	it.config = config
}

func (it *integrationTest) setApps(appNames []string) {
	it.apps = appNames
	appsStr := strings.Join(appNames, ",")

	content := ""
	if appNames != nil {
		content = fmt.Sprintf(`
[pacman]
compose_apps = "%s"
`, appsStr)
	}
	path := filepath.Join(it.tempDir, "z-50-fioctl.toml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		it.t.Fatalf("Failed to write %s toml: %v", path, err)
	}
}

type Target struct {
	Version int
	ID      string
	Apps    []*f.App
	Bad     bool
}

func (t *Target) appsURIs() []string {
	uris := []string{}
	for _, app := range t.Apps {
		uris = append(uris, app.PublishedUri)
	}
	return uris
}

func (it *integrationTest) saveTargetsJson(targets []*Target) {
	targetsStr := []string{}
	currentUTCTime := time.Now().UTC()
	formattedTimeNow := currentUTCTime.Format("2006-01-02T15:04:05Z")
	formattedTimeExpiration := currentUTCTime.AddDate(0, 0, 30).Format("2006-01-02T15:04:05Z")

	for _, target := range targets {
		appsStr := ""
		for _, app := range target.Apps {
			appsStr += fmt.Sprintf(`"%s": {"uri": "%s"},`, app.Name, app.PublishedUri)
		}
		appsStr = strings.TrimSuffix(appsStr, ",")

		targetStr := fmt.Sprintf(`
      "%s": {
        "custom": {
          "arch": "x86_64",
          "containers-sha": "5c0d79ba139344e856f844299bcda57880e1ec46",
          "createdAt": "%s",
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
            "main"
          ],
          "targetFormat": "OSTREE",
          "updatedAt": "%s",
          "version": "%d"
        },
        "hashes": {
          "sha256": "1cb4e739e9fc8d823f8bc6bd0a4c4da144d54f0bee9d49165026ae3290e4e282"
        },
        "length": 0
      }`, target.ID, formattedTimeNow, appsStr, formattedTimeNow, target.Version)

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
    "expires": "%s",
    "targets": {
	  %s
    },
    "version": 352
  }
}
`, formattedTimeExpiration, targetsJson)

	outDir := it.tempDir + "/http_get/repo/"
	if err := os.MkdirAll(outDir, 0o700); err != nil && !os.IsExist(err) {
		it.t.Fatalf("Unable to create http_get/repo dir: %v", err)
	}

	if err := os.WriteFile(outDir+"targets.json", []byte(fullJson), 0o600); err != nil {
		it.t.Fatalf("Unable to write targets.json: %v", err)
	}
}

func (it *integrationTest) genNewTarget(version int, numberOfApps int, portOffset int, badApps bool, nameSuffix string) *Target {
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
	if badApps {
		appComposeDef = strings.ReplaceAll(appComposeDef, "sh -c", "badcommand")
	}
	apps := []*f.App{}

	for i := 1; i <= numberOfApps; i++ {
		port := 8080 + i + portOffset
		app := f.NewApp(it.t, fmt.Sprintf(appComposeDef, port), fmt.Sprintf("app-%d", i))
		app.Publish(it.t)

		app.Pull(it.t)
		defer app.Remove(it.t)
		// app.CheckFetched(t)
		apps = append(apps, app)
	}
	return &Target{
		Version: version,
		ID:      fmt.Sprintf("intel-corei7-64-lmp-%d%s", version, nameSuffix),
		Apps:    apps,
		Bad:     badApps,
	}
}

func runningAppsURIs(status *status.CurrentStatus) []string {
	runningApps := []string{}
	for _, app := range status.AppStatuses {
		if app.Running {
			runningApps = append(runningApps, app.URI)
		}
	}
	return runningApps
}

func filterAppsByName(expectedApps []string, appsNames []string) []string {
	if appsNames == nil {
		return expectedApps
	}
	filtered := []string{}
	for _, expectedApp := range expectedApps {
		// example: registry:5000/factory/app-1@sha256:f872bba0a3738f7e3af34aead338ac142e7df4ea6fe330345931cbcb20cbbcf8
		l := strings.Split(expectedApp, "/")
		appName := strings.Split(l[len(l)-1], "@")[0]
		if slices.Contains(appsNames, appName) {
			filtered = append(filtered, expectedApp)
		}
	}

	return filtered
}

func (it *integrationTest) checkStatus(targetID string, expectedApps []string, filterApps bool) {
	status, err := status.GetCurrentStatus(it.ctx, it.config.ComposeConfig())
	runningApps := runningAppsURIs(status)
	if filterApps {
		expectedApps = filterAppsByName(expectedApps, it.apps)
	}
	assert.ElementsMatch(it.t, expectedApps, runningApps)
	checkErr(it.t, err)
	if status.TargetID != targetID {
		it.t.Fatalf("Current target %s does not match expected target %s", status.TargetID, targetID)
	}
}

func checkErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type integrationTest struct {
	t        *testing.T
	tempDir  string
	config   *cfg.Config
	ctx      context.Context
	gwClient *client.GatewayClient
	apiOpts  []api.UpdateOpt
	apps     []string
}

func newIntegrationTest(t *testing.T) *integrationTest {
	cleanupDockerImages()
	tempDir := t.TempDir()
	config := createMockConfig(t, tempDir)
	gwClient, err := client.NewGatewayClient(config, nil, "", client.WithHttpOperations(mockHttpOperations{config: config, tempDir: tempDir}))
	checkErr(t, err)

	return &integrationTest{
		t:        t,
		tempDir:  tempDir,
		config:   config,
		ctx:      context.Background(),
		gwClient: gwClient,
		apiOpts:  []api.UpdateOpt{api.WithTUF(false), api.WithGatewayClient(gwClient)},
	}
}

func execCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func cleanupDockerImages() {
	cmd := []string{
		"docker stop $(docker ps -aq) 2> /dev/null",
		"docker rm -f $(docker ps -aq) 2> /dev/null",
		"docker image rm -f $(docker images -aq) 2> /dev/null",
	}

	for _, c := range cmd {
		_ = execCommand("sh", "-c", c)
	}
}
