package integration_tests

import (
	"testing"

	"github.com/foundriesio/fioconfig/transport"
)

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
