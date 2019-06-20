
TARGET=transcribe-server

.PHONY: default

default:
	go build -o $(TARGET) ./cmd/transcribe-server/main.go