package main

import (
	"fmt"

	"github.com/foundriesio/fioconfig/transport"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func runDockerCredsHelper() int {
	client, err := transport.CreateClient(config.TomlConfig())
	cobra.CheckErr(err)

	server := config.TomlConfig().Get("tls.server")
	credsUrl := server + "/hub-creds/"
	log.Debug().Str("url", credsUrl).Msg("Getting credentials")

	res, err := transport.HttpGet(client, credsUrl, nil)
	cobra.CheckErr(err)

	if res.StatusCode != 200 {
		cobra.CheckErr(fmt.Errorf("unable to get fioup credentials. HTTP_%d: %s", res.StatusCode, res.String()))
	}
	fmt.Println(res.String())
	return 0
}
