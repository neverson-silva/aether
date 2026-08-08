package client

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	Server string
	Token  string
	HTTP   *http.Client
}

type Config struct {
	Server string `json:"server"`
	Token  string `json:"token"`
}

func LoadConfig() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func SaveConfig(c *Config) error {
	path := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func ConfigPath() string {
	if v := os.Getenv("AETHER_CONFIG"); v != "" {
		return v
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".aether-cli.json"
	}
	return filepath.Join(home, ".config", "aether", "config.json")
}

func New(cfg *Config) *Client {
	server := strings.TrimRight(cfg.Server, "/")
	if server == "" {
		server = "http://127.0.0.1:8080"
	}
	return &Client{
		Server: server,
		Token:  cfg.Token,
		HTTP:   &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) request(method, path string, body any, out any) (int, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, c.Server+path, reader)
	if err != nil {
		return 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		json.Unmarshal(data, &errResp)
		if errResp.Error == "" {
			errResp.Error = strings.TrimSpace(string(data))
		}
		return resp.StatusCode, errors.New(errResp.Error)
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return resp.StatusCode, err
		}
	}
	return resp.StatusCode, nil
}

func (c *Client) Login(email, password string) (*Config, error) {
	var resp struct {
		Token string `json:"token"`
	}
	status, err := c.request("POST", "/api/v1/auth/login", map[string]string{
		"email": email, "password": password,
	}, &resp)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("login falhou: status %d", status)
	}
	cfg := &Config{Server: c.Server, Token: resp.Token}
	return cfg, nil
}

func (c *Client) GetJSON(path string, out any) error {
	status, err := c.request("GET", path, nil, out)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("status %d", status)
	}
	return nil
}

func (c *Client) PostJSON(path string, body, out any) error {
	status, err := c.request("POST", path, body, out)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("status %d", status)
	}
	return nil
}

func (c *Client) PutJSON(path string, body, out any) error {
	status, err := c.request("PUT", path, body, out)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("status %d", status)
	}
	return nil
}

func (c *Client) Delete(path string) error {
	status, err := c.request("DELETE", path, nil, nil)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("status %d", status)
	}
	return nil
}

func (c *Client) Logs(appID string, follow bool) error {
	path := "/api/v1/apps/" + appID + "/logs"
	if follow {
		path += "?follow=1"
	}
	req, err := http.NewRequest("GET", c.Server+path, nil)
	if err != nil {
		return err
	}
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			fmt.Println(strings.TrimPrefix(line, "data: "))
		}
	}
	return scanner.Err()
}
