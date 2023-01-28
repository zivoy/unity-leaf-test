package backend

import (
	"fmt"
	"image/color"
)

func colourToHex(colour color.Color) string {
	r, g, b, _ := colour.RGBA()
	return fmt.Sprintf("#%02x%02x%02x", r&0xff, g&0xff, b&0xff)
}

func colourFromHex(colour string) color.RGBA {
	c := color.RGBA{A: 0xff}
	var err error
	switch len(colour) {
	case 6, 3:
		return colourFromHex("#" + colour)
	case 7:
		_, err = fmt.Sscanf(colour, "#%02x%02x%02x", &c.R, &c.G, &c.B)
	case 4:
		_, err = fmt.Sscanf(colour, "#%1x%1x%1x", &c.R, &c.G, &c.B)
		// Double the hex digits:
		c.R *= 17
		c.G *= 17
		c.B *= 17
	default:
		err = fmt.Errorf("invalid length")

	}
	if err != nil {
		c.R = 0xff
		c.G = 0
		c.B = 0
	}
	return c
}
