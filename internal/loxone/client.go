package loxone

import (
	"fmt"
	"net"
	"net/http"
	"time"
)

type Client struct {
	ms        *LBMiniserver
	transport string
	udpPort   int
	http      *http.Client
}

func NewClient(ms *LBMiniserver, transport string, udpPort int) *Client {
	return &Client{
		ms:        ms,
		transport: transport,
		udpPort:   udpPort,
		http: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *Client) Send(viName, value string) error {
	switch c.transport {
	case "udp":
		return c.sendUDP(viName, value)
	default:
		return c.sendHTTP(viName, value)
	}
}

func (c *Client) sendHTTP(viName, value string) error {
	port := c.ms.Port
	if port == "" {
		port = "80"
	}
	url := fmt.Sprintf("http://%s:%s/dev/sps/io/%s/%s", c.ms.IPAddress, port, viName, value)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.ms.Admin, c.ms.Pass)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("loxone HTTP %d for %s", resp.StatusCode, viName)
	}
	return nil
}

func (c *Client) sendUDP(viName, value string) error {
	addr := fmt.Sprintf("%s:%d", c.ms.IPAddress, c.udpPort)
	conn, err := net.Dial("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	msg := fmt.Sprintf("%s=%s\r\n", viName, value)
	_, err = conn.Write([]byte(msg))
	return err
}

func (c *Client) BaseURL() string {
	port := c.ms.Port
	if port == "" {
		port = "80"
	}
	return fmt.Sprintf("http://%s:%s", c.ms.IPAddress, port)
}

func (c *Client) Credentials() (string, string) {
	return c.ms.Admin, c.ms.Pass
}
