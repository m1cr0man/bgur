.PHONY: default
default: generate build

.PHONY: generate
generate:
	cat pkg/oauth2/callbackpage.html | ./embed.sh oauth2 CallbackPage --compress > pkg/oauth2/callbackpage.go

.PHONY: build
build:
	go build -o bgur cmd/bgur/main.go
