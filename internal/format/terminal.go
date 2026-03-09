package format

import (
	"sync"

	"github.com/BourgeoisBear/rasterm"
)

// ImageProtocol represents the best available image rendering protocol.
type ImageProtocol int

const (
	ProtoHalfBlock ImageProtocol = iota // Universal fallback
	ProtoKitty                          // Kitty graphics protocol
	ProtoIterm                          // iTerm2 inline images
	ProtoSixel                          // Sixel graphics
)

var (
	detectedProtocol ImageProtocol
	detectOnce       sync.Once
)

// DetectImageProtocol returns the best image protocol available in the current terminal.
func DetectImageProtocol() ImageProtocol {
	detectOnce.Do(func() {
		detectedProtocol = detectProtocol()
	})
	return detectedProtocol
}

func detectProtocol() ImageProtocol {
	if rasterm.IsKittyCapable() {
		return ProtoKitty
	}
	if rasterm.IsItermCapable() {
		return ProtoIterm
	}
	if ok, _ := rasterm.IsSixelCapable(); ok {
		return ProtoSixel
	}
	return ProtoHalfBlock
}
