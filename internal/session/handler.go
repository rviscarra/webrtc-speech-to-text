package session

import (
	"encoding/json"
	"net/http"

	"github.com/rviscarra/webrtc-speech-to-text/internal/rtc"
)

// MakeHandler returns an HTTP handler for the session service
func MakeHandler(webrtc rtc.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		dec := json.NewDecoder(r.Body)
		req := newSessionRequest{}

		if err := dec.Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		peer, err := webrtc.CreatePeerConnection()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		answer, err := peer.ProcessOffer(req.Offer)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		payload, err := json.Marshal(newSessionResponse{
			Answer: answer,
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		w.Write(payload)
	})
	return mux
}
