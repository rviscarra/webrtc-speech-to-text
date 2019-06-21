package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rviscarra/webrtc-speech-to-text/internal/rtc"
	"github.com/rviscarra/webrtc-speech-to-text/internal/session"
	"github.com/rviscarra/webrtc-speech-to-text/internal/transcribe"
)

const (
	httpDefaultPort   = "9000"
	defaultStunServer = "stun:stun.l.google.com:19302"
)

func main() {

	httpPort := flag.String("http.port", httpDefaultPort, "HTTP listen port")
	stunServer := flag.String("stun.server", defaultStunServer, "STUN server URL (stun:)")
	speechCred := flag.String("google.cred", "", "Google Speech credentials file")
	flag.Parse()

	if *speechCred == "" {
		log.Fatal("You need to specify the Google credentials file")
	}

	var tr transcribe.Service
	ctx := context.Background()
	tr, err := transcribe.NewGoogleSpeech(ctx, *speechCred)

	var webrtc rtc.Service
	webrtc = rtc.NewPionRtcService(*stunServer, tr)
	// webrtc = rtc.NewLoggingService(webrtc)

	// Endpoint to create a new speech to text session
	http.Handle("/session", session.MakeHandler(webrtc))

	// Serve static assets
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/index.html")
	})
	http.Handle("/static/", http.StripPrefix("/static", http.FileServer(http.Dir("./web"))))

	errors := make(chan error, 2)
	go func() {
		log.Printf("Starting signaling server on port %s", *httpPort)
		errors <- http.ListenAndServe(fmt.Sprintf(":%s", *httpPort), nil)
	}()

	go func() {
		interrupt := make(chan os.Signal)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
		errors <- fmt.Errorf("Received %v signal", <-interrupt)
	}()

	err = <-errors
	log.Printf("%s, exiting.", err)
}
