package loxone

import (
	"strings"
	"testing"
)

func TestSendRejectsUnsupportedTransport(t *testing.T) {
	c := NewClient(&LBMiniserver{}, "mqtt", 7777)
	err := c.Send("test", "1")
	if err == nil || !strings.Contains(err.Error(), "unsupported Loxone transport") {
		t.Fatalf("expected unsupported transport error, got %v", err)
	}
}

func TestBaseURLJoinsIPv6HostAndPort(t *testing.T) {
	c := NewClient(&LBMiniserver{IPAddress: "::1", Port: "1234"}, "http", 7777)
	if got, want := c.BaseURL(), "http://[::1]:1234"; got != want {
		t.Fatalf("BaseURL() = %q, want %q", got, want)
	}
}

func TestBaseURLJoinsIPv4HostAndPort(t *testing.T) {
	c := NewClient(&LBMiniserver{IPAddress: "192.0.2.10", Port: "8080"}, "http", 7777)
	if got, want := c.BaseURL(), "http://192.0.2.10:8080"; got != want {
		t.Fatalf("BaseURL() = %q, want %q", got, want)
	}
}
