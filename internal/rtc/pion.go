package rtc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/pion/webrtc/v2"
	"github.com/rviscarra/webrtc-speech-to-text/internal/transcribe"
)

// PionPeerConnection TODO
type PionPeerConnection struct {
	pc *webrtc.PeerConnection
}

// PionRtcService TODO
type PionRtcService struct {
	stunServer  string
	transcriber transcribe.Service
}

// NewPionService TODO
func NewPionService(stun string, transcriber transcribe.Service) Service {
	return &PionRtcService{
		stunServer:  stun,
		transcriber: transcriber,
	}
}

// ProcessOffer TODO
func (p *PionPeerConnection) ProcessOffer(offer string) (string, error) {
	err := p.pc.SetRemoteDescription(webrtc.SessionDescription{
		SDP:  offer,
		Type: webrtc.SDPTypeOffer,
	})
	if err != nil {
		return "", err
	}

	answer, err := p.pc.CreateAnswer(nil)
	if err != nil {
		return "", err
	}
	err = p.pc.SetLocalDescription(answer)
	if err != nil {
		return "", err
	}
	return answer.SDP, nil
}

// Close TODO
func (p *PionPeerConnection) Close() error {
	return p.pc.Close()
}

func (pi *PionRtcService) handleAudioTrack(track *webrtc.Track, dc *webrtc.DataChannel) error {
	decoder, err := newDecoder()
	if err != nil {
		return err
	}
	trStream, err := pi.transcriber.CreateStream()
	if err != nil {
		return err
	}
	defer func() {
		err := trStream.Close()
		if err != nil {
			log.Printf("Error closing stream %v", err)
			return
		}
		for result := range trStream.Results() {
			log.Printf("Result: %v", result)
			msg, err := json.Marshal(result)
			if err != nil {
				continue
			}
			err = dc.Send(msg)
			if err != nil {
				fmt.Printf("DataChannel error: %v", err)
			}
		}
		dc.Close()
	}()

	errs := make(chan error, 2)
	audioStream := make(chan []byte)
	response := make(chan bool)
	timer := time.NewTimer(5 * time.Second)
	go func() {
		for {
			packet, err := track.ReadRTP()
			timer.Reset(1 * time.Second)
			if err != nil {
				timer.Stop()
				if err == io.EOF {
					close(audioStream)
					return
				}
				errs <- err
				return
			}
			audioStream <- packet.Payload
			<-response
		}
	}()
	err = nil
	for {
		select {
		case audioChunk := <-audioStream:
			payload, err := decoder.decode(audioChunk)
			response <- true
			if err != nil {
				return err
			}
			_, err = trStream.Write(payload)
			if err != nil {
				return err
			}
		case <-timer.C:
			return fmt.Errorf("Read operation timed out")
		case err = <-errs:
			log.Printf("Unexpected error reading track %s: %v", track.ID(), err)
			return err
		}
	}
}

// CreatePeerConnection TODO
func (pi *PionRtcService) CreatePeerConnection() (PeerConnection, error) {
	pcconf := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			webrtc.ICEServer{
				URLs: []string{pi.stunServer},
			},
		},
		SDPSemantics: webrtc.SDPSemanticsUnifiedPlanWithFallback,
	}
	pc, err := webrtc.NewPeerConnection(pcconf)
	if err != nil {
		return nil, err
	}

	dataChan := make(chan *webrtc.DataChannel)

	pc.OnDataChannel(func(dc *webrtc.DataChannel) {
		dataChan <- dc
	})

	pc.OnTrack(func(track *webrtc.Track, r *webrtc.RTPReceiver) {
		if track.Codec().Name == "opus" {
			log.Printf("Received audio (%s) track, id = %s\n", track.Codec().Name, track.ID())
			err := pi.handleAudioTrack(track, <-dataChan)
			if err != nil {
				log.Printf("Error reading track (%s): %v\n", track.ID(), err)
			}
		}
	})

	pc.OnICEConnectionStateChange(func(connState webrtc.ICEConnectionState) {
		log.Printf("Connection state: %s \n", connState.String())
	})

	_, err = pc.AddTransceiver(webrtc.RTPCodecTypeAudio, webrtc.RtpTransceiverInit{
		Direction: webrtc.RTPTransceiverDirectionRecvonly,
	})
	if err != nil {
		log.Printf("Can't add transceiver: %s", err)
		return nil, err
	}

	return &PionPeerConnection{
		pc: pc,
	}, nil
}
