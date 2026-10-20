package loxone

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
)

// Loxone reports command errors in an LL envelope, often with HTTP 200.
func responseCode(r io.Reader) (string, error) {
	body, err := io.ReadAll(io.LimitReader(r, 1<<20))
	if err != nil {
		return "", err
	}
	body = bytes.TrimSpace(body)
	var code string
	if len(body) > 0 && body[0] == '<' {
		var result struct {
			XMLName xml.Name `xml:"LL"`
			Code    string   `xml:"Code,attr"`
		}
		err = xml.Unmarshal(body, &result)
		code = result.Code
	} else {
		var result struct {
			LL struct {
				Code json.RawMessage `json:"Code"`
			} `json:"LL"`
		}
		err = json.Unmarshal(body, &result)
		code = string(bytes.Trim(result.LL.Code, "\""))
	}
	if err != nil || code == "" {
		return "", fmt.Errorf("ungültige Miniserver-Antwort (LL-Code fehlt)")
	}
	return code, nil
}
