package integration_tests

import (
	"context"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/foundriesio/composeapp/test/fixtures"
	"github.com/foundriesio/fioconfig/transport"
)

type (
	Server struct {
		root string
		srv  *http.Server
		ln   net.Listener
	}
)

func NewServer(rootDir string) (*Server, error) {
	return &Server{root: rootDir}, nil
}

func (s *Server) Start() error {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.ln = ln

	s.srv = &http.Server{
		Handler: http.HandlerFunc(s.handle),
	}

	go func() {
		_ = s.srv.Serve(ln)
	}()

	return nil
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.srv.Shutdown(ctx)
}

func (s *Server) Addr() string {
	return "http://" + s.ln.Addr().String()
}

func (s *Server) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Path examples:
	// - /v2/factory/app-1/blobs/sha256:e6c3cae71e67f5c98ffb22a65ca771858b14c1dac093ab66235b787181ad1760
	// - /v2/factory/app-1/manifests/sha256:e6c3cae71e67f5c98ffb22a65ca771858b14c1dac093ab66235b787181ad1760
	hash, _ := digestFromPath(r.URL.Path)
	fullPath := filepath.Join(s.root, hash)

	if !strings.HasPrefix(fullPath, s.root) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	file, err := os.Open(fullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		http.NotFound(w, r)
		return
	}

	// DO NOT special-case HEAD
	http.ServeContent(
		w,
		r,
		stat.Name(),
		stat.ModTime(),
		file,
	)
}

func digestFromPath(path string) (string, bool) {
	const prefix = "sha256:"

	// Get last path segment
	i := strings.LastIndex(path, "/")
	if i == -1 {
		return "", false
	}

	last := path[i+1:]
	if !strings.HasPrefix(last, prefix) {
		return "", false
	}

	return strings.TrimPrefix(last, prefix), true
}

func TestUpdateThroughProxy_EndpointNotFound(t *testing.T) {
	it := newIntegrationTest(t,
		WithProxyAuthURL("https://example.com/apps-proxy-url"),
		WithProxyHandler(func() (*transport.HttpRes, error) {
			res := transport.HttpRes{
				StatusCode: 404,
				Body:       []byte(`{"error":"not found"}`),
			}
			return &res, nil
		}))
	target1 := it.genNewTarget(100, 1, 50, false, "")
	targets := []*Target{target1}
	it.saveTargetsJson(targets)
	it.testUpdateTo(target1, targets)
}

func TestUpdateThroughProxy_InvalidProxyURL(t *testing.T) {
	it := newIntegrationTest(t,
		WithProxyAuthURL("https://example.com/apps-proxy-url"),
		WithProxyHandler(func() (*transport.HttpRes, error) {
			res := transport.HttpRes{
				StatusCode: 201,
				Body:       []byte("foobar"),
			}
			return &res, nil
		}))
	target1 := it.genNewTarget(100, 1, 50, false, "")
	targets := []*Target{target1}
	it.saveTargetsJson(targets)
	it.testUpdateTo(target1, targets)
}

func TestUpdateThroughProxy_FetchApps(t *testing.T) {
	srv, err := NewServer(filepath.Join(fixtures.AppStoreRoot, "blobs", "sha256"))
	checkErr(t, err)
	err = srv.Start()
	checkErr(t, err)
	defer func() {
		err := srv.Stop()
		checkErr(t, err)
	}()

	it := newIntegrationTest(t,
		WithProxyAuthURL("https://example.com/apps-proxy-url"),
		WithProxyHandler(func() (*transport.HttpRes, error) {
			res := transport.HttpRes{
				StatusCode: 201,
				Body:       []byte(srv.Addr() + "/"),
			}
			return &res, nil
		}))
	target1 := it.genNewTarget(100, 1, 50, false, "")
	targets := []*Target{target1}
	it.saveTargetsJson(targets)

	app := target1.Apps[0]
	app.Pull(t)
	defer app.Remove(t)

	it.testUpdateTo(target1, targets)
}
