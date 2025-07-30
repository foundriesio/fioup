package main

import (
	"fmt"
	"github.com/foundriesio/fioconfig/sotatoml"
	"github.com/foundriesio/fioconfig/transport"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"net/url"
)

type (
	httpOptions struct {
	}
)

func init() {
	httpCmd := &cobra.Command{
		Use:   "http <get> <endpoint or full URL> [flags]",
		Short: "Perform an HTTP request to a server by using mTLS credentials",
		Long: `Perform an HTTP request to a server by using mTLS credentials.
The command supports HTTP requests to a specified endpoint or full URL.
By default, it uses the server base URL defined in the configuration file.`,
		Args:   cobra.MinimumNArgs(1),
		Hidden: true,
	}

	opts := httpOptions{}

	getCmd := &cobra.Command{
		Use:  "get <endpoint or URL>",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			doGet(cmd, args, &opts)
		},
	}
	// TODO: Add support for other HTTP methods like POST, PUT, PATCH, DELETE
	for _, cmd := range []*cobra.Command{getCmd} {
		httpCmd.AddCommand(cmd)
	}

	rootCmd.AddCommand(httpCmd)
}

func doGet(cmd *cobra.Command, args []string, opts *httpOptions) {
	url, err := getUrl(config, args[0])
	DieNotNil(err, "failed to parse URL or endpoint")

	client := transport.CreateClient(config)
	resp, err := transport.HttpGet(client, url, nil)
	if err != nil || resp.StatusCode != 200 {
		if err == nil {
			log.Printf("Error:  %s: %s\n", url, resp)
			err = fmt.Errorf("HTTP request failed: %s", resp)
		}
		DieNotNil(err)
	}
	fmt.Print(resp)
}

func getUrl(cfg *sotatoml.AppConfig, endpointOrUrl string) (string, error) {
	url, err := url.Parse(endpointOrUrl)
	if err != nil {
		return "", fmt.Errorf("invalid URL or endpoint: %w", err)
	}
	if url.Scheme == "https" {
		return url.String(), nil
	}

	server := cfg.Get("tls.server")
	url, err = url.Parse(server)
	if err != nil {
		return "", fmt.Errorf("invalid server base URL in config: %w", err)
	}
	url.Path += endpointOrUrl
	return url.String(), nil
}
