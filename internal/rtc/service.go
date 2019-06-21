package rtc

import (
	"io"
)

// PeerConnection Represents a WebRTC connection to a single peer
type PeerConnection interface {
	io.Closer
	ProcessOffer(offer string) (string, error)
}

// Service WebRTC service
type Service interface {
	CreatePeerConnection() (PeerConnection, error)
}
