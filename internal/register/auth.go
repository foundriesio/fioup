// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package register

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

const (
	DEVICE_API = "https://api.foundries.io/ota/devices/"
	OAUTH_API  = "https://app.foundries.io/oauth"
)

type OauthCallback interface {
	ShowAuthInfo(deviceUuid, userCode, url string, expiresMinutes int)
	Tick()
}

type HttpHeaders map[string]string

func respToErr(code int, resp map[string]any) error {
	var buf strings.Builder
	fmt.Fprintf(&buf, "HTTP_%d\n", code)

	if data, ok := resp["data"].(string); ok && len(data) > 0 {
		fmt.Fprintln(&buf, data)
	}
	for k, v := range resp {
		fmt.Fprintf(&buf, " | %s: %v\n", k, v)
	}

	return errors.New(buf.String())
}

// Returns the access_token on success, and empty string on errors
func getOauthToken(cb OauthCallback, factory, deviceUUID string) (string, error) {
	env := os.Getenv(ENV_OAUTH_BASE)
	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}
	url := OAUTH_API
	if env != "" {
		url = env
	}

	data := fmt.Sprintf("client_id=%s&scope=%s:devices:create", deviceUUID, factory)
	resp, code, err := httpPost(url+"/authorization/device/", headers, data)
	if err != nil {
		return "", err
	} else if code != 200 {
		return "", fmt.Errorf("unable to create device authorization request: %w", respToErr(code, resp))
	}

	expiresMinutes := int(resp["expires_in"].(float64)) / 60
	uc := resp["user_code"].(string)
	uri := resp["verification_uri"].(string)
	cb.ShowAuthInfo(deviceUUID, uc, uri, expiresMinutes)

	data = fmt.Sprintf(
		"grant_type=urn:ietf:params:oauth:grant-type:device_code&device_code=%s&client_id=%s&scope=%s:devices:create",
		resp["device_code"], deviceUUID, factory,
	)
	interval := int(resp["interval"].(float64))

	log.Debug().Str("data", data).Msg("oauth data")
	for {
		tokenResp, code, err := httpPost(url+"/token/", headers, data)
		if err == nil && code == 200 {
			return tokenResp["access_token"].(string), nil
		}
		if code != 400 {
			log.Warn().Int("code", code).Msg("HTTP error...")
			time.Sleep(2 * time.Second)
			continue
		}
		if tokenResp["error"] == "authorization_pending" {
			cb.Tick()
			time.Sleep(time.Duration(interval) * time.Second)
		} else {
			return "", fmt.Errorf("unable to authorize device: %w", respToErr(code, resp))
		}
	}
}

func httpPost(url string, headers map[string]string, data string) (map[string]interface{}, int, error) {
	req, err := http.NewRequest("POST", url, strings.NewReader(data))
	if err != nil {
		return nil, 0, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Err(closeErr).Msgf("failed to close resp.Body")
		}
	}()
	body, _ := io.ReadAll(resp.Body)
	var jsonResp map[string]interface{}

	err = json.Unmarshal(body, &jsonResp)
	if err != nil {
		log.Err(err).Msgf("failed to unmarshal response body: %s", string(body))
		return nil, resp.StatusCode, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return jsonResp, resp.StatusCode, nil
}

func AuthGetHttpHeaders(opt *RegisterOptions, cb OauthCallback) (HttpHeaders, error) {
	headers := map[string]string{"Content-type": "application/json"}
	if opt.ApiToken != "" {
		headers[opt.ApiTokenHeader] = opt.ApiToken
		return headers, nil
	}
	log.Debug().Msg("Foundries providing auth token")
	token, err := getOauthToken(cb, opt.Factory, opt.UUID)
	if err != nil {
		return nil, err
	}
	tokenBase64 := base64.StdEncoding.EncodeToString([]byte(token))
	headers["Authorization"] = "Bearer " + tokenBase64
	return headers, nil
}

// Register device using the oauth token. Token need "devices:create" scope
func AuthRegisterDevice(headers HttpHeaders, device map[string]interface{}) (map[string]interface{}, error) {
	api := os.Getenv(ENV_DEVICE_API)
	if api == "" {
		api = DEVICE_API
	}
	data, _ := json.MarshalIndent(device, "", "  ")

	log.Debug().
		Str("name", device["name"].(string)).
		Str("url", api).
		Str("csr", string(data)).
		Msg("Registering device")
	jsonResp, code, err := httpPost(api, headers, string(data))
	if code != 201 || err != nil {
		return nil, fmt.Errorf("unable to create device: %w", respToErr(code, jsonResp))
	}
	return jsonResp, nil
}

func AuthPingServer() error {
	api := os.Getenv(ENV_DEVICE_API)
	if api == "" {
		api = DEVICE_API
	}
	log.Debug().Str("url", api).Msg("Using DEVICE_API")
	resp, err := http.Get(api)
	if err != nil || resp.StatusCode > 500 {
		return fmt.Errorf("ping failed %w", err)
	}
	return nil
}
