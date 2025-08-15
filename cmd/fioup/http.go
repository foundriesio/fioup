// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"fmt"
	"github.com/foundriesio/fioconfig/sotatoml"
	"github.com/foundriesio/fioconfig/transport"
	"github.com/spf13/cobra"
	"net/http"
	"net/url"
)

func init() {
	httpCmd := &cobra.Command{
		Use:   "http <get> <endpoint or full URL> [flags]",
		Short: "Perform an HTTP request to a server by using mTLS credentials",
		Long: `Perform an HTTP request to a server by using mTLS credentials.
The command supports HTTP requests to a specified endpoint or full URL.
By default, it uses the server base URL defined in the configuration file. For example:

fioup http get hub-creds/ - get auth credentials to authenticate at the registry auth server
fioup http post https://ostree.foundries.io:8443/ostree/download-urls - get download URLs along with auth tokens to fetch ostree commit from the google storage`,
		Args:   cobra.MinimumNArgs(1),
		Hidden: true,
	}

	getCmd := &cobra.Command{
		Use:  "get <endpoint or URL>",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			doRequest(args[0], func(client *http.Client, url string) (*transport.HttpRes, error) {
				return transport.HttpGet(client, url, nil)
			})
		},
	}
	postCmd := &cobra.Command{
		Use:  "post <endpoint or URL>",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			doRequest(args[0], func(client *http.Client, url string) (*transport.HttpRes, error) {
				return transport.HttpPost(client, url, nil)
			})
		},
	}
	// TODO: Add support for other HTTP methods like POST, PUT, PATCH, DELETE
	for _, cmd := range []*cobra.Command{getCmd, postCmd} {
		httpCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(httpCmd)
}

func doRequest(endpointOrUrl string, request func(*http.Client, string) (*transport.HttpRes, error)) {
	url := getUrl(config, endpointOrUrl)
	client, err := transport.CreateClient(config)
	DieNotNil(err, "failed to create HTTPS client with the given configuration")
	processResponse(request(client, url))
}

func getUrl(cfg *sotatoml.AppConfig, endpointOrUrl string) string {
	url, err := url.Parse(endpointOrUrl)
	DieNotNil(err, "failed to parse URL or endpoint "+endpointOrUrl)
	// If the URL is already a full URL with a scheme, return it directly
	if url.Scheme == "https" {
		return url.String()
	}
	// Otherwise, construct the full URL using the server base URL from the config
	server := cfg.Get("tls.server")
	url, err = url.Parse(server)
	DieNotNil(err, "failed to parse server base URL "+server)
	url.Path += endpointOrUrl
	return url.String()
}

func processResponse(resp *transport.HttpRes, err error) {
	if err != nil || resp.StatusCode != 200 {
		if err == nil {
			err = fmt.Errorf("HTTP request failed: %s", resp)
		}
		DieNotNil(err)
	}
	fmt.Print(resp)
}
