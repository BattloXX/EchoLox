package hue

// BriToPct converts Hue brightness (0-254) to percent (0-100)
func BriToPct(bri int) int {
	if bri <= 0 {
		return 0
	}
	pct := (bri * 100) / 254
	if pct > 100 {
		pct = 100
	}
	return pct
}

// HueTo360 converts Hue hue value (0-65535) to degrees (0-360)
func HueTo360(h int) int {
	return (h * 360) / 65535
}

// SatToPct converts Hue saturation (0-254) to percent (0-100)
func SatToPct(sat int) int {
	if sat <= 0 {
		return 0
	}
	return (sat * 100) / 254
}
