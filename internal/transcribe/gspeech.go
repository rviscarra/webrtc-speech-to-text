package transcribe

import (
	"context"
	"fmt"
	"io"
	"log"

	speech "cloud.google.com/go/speech/apiv1"
	"google.golang.org/api/option"
	speechpb "google.golang.org/genproto/googleapis/cloud/speech/v1"
)

// GoogleTranscriber is the implementation of the transcribe.Service,
// hold a pointer to the Google Speech client
type GoogleTranscriber struct {
	speechClient *speech.Client
	ctx          context.Context
}

// GoogleTrStream implements the transcribe.Stream interface,
// it should map one to one with the audio stream coming from the client
type GoogleTrStream struct {
	stream  speechpb.Speech_StreamingRecognizeClient
	results chan Result
}

// CreateStream creates a new transcription stream
func (t *GoogleTranscriber) CreateStream() (Stream, error) {
	stream, err := t.speechClient.StreamingRecognize(t.ctx)
	if err != nil {
		return nil, err
	}

	// Send the initial configuration message.
	if err := stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_StreamingConfig{
			StreamingConfig: &speechpb.StreamingRecognitionConfig{
				Config: &speechpb.RecognitionConfig{
					Encoding:          speechpb.RecognitionConfig_LINEAR16,
					SampleRateHertz:   48000,
					LanguageCode:      "en-US",
					AudioChannelCount: 1,
				},
			},
		},
	}); err != nil {
		return nil, err
	}

	return &GoogleTrStream{
		stream:  stream,
		results: make(chan Result),
	}, nil
}

// Results returns a channel that will receive the transcription
// results when they're ready
func (st *GoogleTrStream) Results() <-chan Result {
	return st.results
}

// Close flushes the recognition stream and
// pipes the results to the channel
func (st *GoogleTrStream) Close() error {
	if err := st.stream.CloseSend(); err != nil {
		return err
	}
	resp, err := st.stream.Recv()
	if err != nil && err != io.EOF {
		return err
	}
	if resp == nil {
		close(st.results)
		return nil
	}
	if resp.Error != nil {
		return fmt.Errorf("(Code: %d) %s", resp.Error.GetCode(), resp.Error.GetMessage())
	}

	// This needs to be a Goroutine because our caller may read the results chan
	// after calling this method.
	go func() {
		for _, result := range resp.GetResults() {
			for _, alt := range result.GetAlternatives() {
				log.Printf("%s (%.2f)", alt.GetTranscript(), alt.GetConfidence())
				st.results <- Result{
					Confidence: alt.GetConfidence(),
					Text:       alt.GetTranscript(),
					Final:      result.GetIsFinal(),
				}
			}
		}
		close(st.results)
	}()
	return nil
}

func (st *GoogleTrStream) Write(buffer []byte) (int, error) {
	if err := st.stream.Send(&speechpb.StreamingRecognizeRequest{
		StreamingRequest: &speechpb.StreamingRecognizeRequest_AudioContent{
			AudioContent: buffer,
		},
	}); err != nil {
		return 0, nil
	}
	return len(buffer), nil
}

// NewGoogleSpeech creates a new intances of the transcribe.Service that uses
// Google Speech
func NewGoogleSpeech(ctx context.Context, credentials string) (Service, error) {
	speechClient, err := speech.NewClient(ctx, option.WithCredentialsFile(credentials))
	if err != nil {
		return nil, err
	}
	return &GoogleTranscriber{
		speechClient: speechClient,
		ctx:          ctx,
	}, nil
}
