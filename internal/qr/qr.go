// Package qr renders a string as a terminal-friendly QR code using
// half-block characters so the result is roughly square.
package qr

import qrcode "github.com/skip2/go-qrcode"

// Render returns an ASCII/Unicode QR code suitable for printing to a
// terminal. Each pair of vertical "modules" is collapsed into a single
// character row using ▀ (upper half block) so the QR stays roughly
// square at terminal cell aspect ratio.
func Render(content string) (string, error) {
	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", err
	}
	bits := q.Bitmap()
	// Polarity is inverted relative to the QR (off pixels render as
	// terminal foreground, on pixels as terminal background), which is
	// what `qrencode -t UTF8` and similar tools do — it scans cleanly on
	// dark terminals and is still readable on light ones.
	const (
		empty = "█" // both halves "off" (light) -> full foreground
		upper = "▄" // top "on" (dark), bottom "off" -> lower half foreground
		lower = "▀" // top "off", bottom "on" -> upper half foreground
		full  = " " // both halves "on" -> blank
	)
	var out []byte
	for y := 0; y < len(bits); y += 2 {
		row := bits[y]
		var next []bool
		if y+1 < len(bits) {
			next = bits[y+1]
		}
		for x := 0; x < len(row); x++ {
			topOn := row[x]
			botOn := false
			if next != nil {
				botOn = next[x]
			}
			switch {
			case topOn && botOn:
				out = append(out, full...)
			case topOn:
				out = append(out, upper...)
			case botOn:
				out = append(out, lower...)
			default:
				out = append(out, empty...)
			}
		}
		out = append(out, '\n')
	}
	return string(out), nil
}
