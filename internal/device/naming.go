package device

import (
	"regexp"
	"strings"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

func NormalizeName(name string) string {
	r := strings.NewReplacer(
		"ä", "ae", "ö", "oe", "ü", "ue",
		"Ä", "ae", "Ö", "oe", "Ü", "ue",
		"ß", "ss",
		"à", "a", "á", "a", "â", "a", "ã", "a",
		"è", "e", "é", "e", "ê", "e", "ë", "e",
		"ì", "i", "í", "i", "î", "i", "ï", "i",
		"ò", "o", "ó", "o", "ô", "o", "õ", "o",
		"ù", "u", "ú", "u", "û", "u",
		"ñ", "n", "ç", "c",
	)
	result := r.Replace(name)
	result = strings.ToLower(result)
	result = nonAlphaNum.ReplaceAllString(result, "_")
	result = strings.Trim(result, "_")
	return result
}

func GenerateVirtualInputs(name string, dtype DeviceType) map[string]string {
	n := NormalizeName(name)
	prefix := "echolox_" + n
	m := map[string]string{}
	switch dtype {
	case TypeSwitch:
		m["on"] = prefix + "_on"
		m["off"] = prefix + "_off"
	case TypeDimmer:
		m["on"] = prefix + "_on"
		m["off"] = prefix + "_off"
		m["brightness"] = prefix + "_brightness"
	case TypeColor:
		m["on"] = prefix + "_on"
		m["off"] = prefix + "_off"
		m["brightness"] = prefix + "_brightness"
		m["hue"] = prefix + "_hue"
		m["saturation"] = prefix + "_saturation"
	case TypeScene:
		m["activate"] = prefix + "_activate"
	}
	return m
}
