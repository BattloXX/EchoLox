package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

func main() {
	orange := color.RGBA{0xe8, 0x45, 0x0a, 0xff}
	white := color.RGBA{0xff, 0xff, 0xff, 0xff}

	outDir := "icons"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	for _, spec := range []struct {
		name string
		size int
	}{
		{"EchoLox.png", 48},
		{"EchoLox_icon.png", 48},
	} {
		size := spec.size
		img := image.NewRGBA(image.Rect(0, 0, size, size))

		cx := float64(size) / 2
		cy := float64(size) / 2
		r := float64(size)/2 - 1

		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx := float64(x) - cx + 0.5
				dy := float64(y) - cy + 0.5
				if math.Sqrt(dx*dx+dy*dy) <= r {
					img.Set(x, y, orange)
				}
			}
		}

		// White "E" — three horizontal bars (top, middle shorter, bottom)
		lx := size / 4
		barH := size / 9
		if barH < 2 {
			barH = 2
		}
		top := size / 4
		mid := size/2 - barH/2
		bot := size - size/4 - barH

		for _, bar := range []struct {
			y int
			w int
		}{
			{top, size / 2},
			{mid, size * 3 / 8},
			{bot, size / 2},
		} {
			for dy := 0; dy < barH; dy++ {
				for dx := 0; dx < bar.w; dx++ {
					img.Set(lx+dx, bar.y+dy, white)
				}
			}
		}

		path := filepath.Join(outDir, spec.name)
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		png.Encode(f, img)
		f.Close()
	}
}
