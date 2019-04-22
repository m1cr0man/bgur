.PHONY: default
default: generate build

.PHONY: generate
generate:
	cat pkg/imgur/callbackpage.html | ./embed.sh imgur CallbackPage --compress > pkg/imgur/callbackpage.go

.PHONY: build
build:
	go build -o bgur cmd/bgur/main.go
