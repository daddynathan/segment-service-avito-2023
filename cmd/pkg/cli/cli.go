package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

type Client struct {
	Host string
}

func NewClient(host string) *Client {
	return &Client{Host: host}
}

func (c *Client) Request(method, endpoint, data string) {
	url := c.Host + endpoint
	var req *http.Request
	var err error
	if data != "" {
		req, err = http.NewRequest(method, url, strings.NewReader(data))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		log.Fatal("Failed to create request:", err)
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal("Request failed:", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Failed to read response:", err)
	}
	var prettyJSON map[string]interface{}
	if err := json.Unmarshal(body, &prettyJSON); err == nil {
		prettyBody, _ := json.MarshalIndent(prettyJSON, "", "  ")
		fmt.Printf("Status: %s\n", resp.Status)
		fmt.Println("Response:")
		fmt.Println(string(prettyBody))
	} else {
		fmt.Printf("Status: %s\n", resp.Status)
		fmt.Printf("Response: %s\n", string(body))
	}
}
