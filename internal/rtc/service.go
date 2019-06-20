package rtc

import (
	"io"
)

// PeerConnection TODO
type PeerConnection interface {
	io.Closer
	ProcessOffer(offer string) (string, error)
}

// Service TODO
type Service interface {
	CreatePeerConnection() (PeerConnection, error)
}
