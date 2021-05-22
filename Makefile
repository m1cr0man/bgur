.PHONY: default
default: generate build

.PHONY: generate
generate:
	cat pkg/oauth2/callbackpage.html | ./embed.sh oauth2 CallbackPage --compress > pkg/oauth2/callbackpage.go

bgur:
	go build -o bgur cmd/bgur/main.go

upload_tree:
	go build -o upload_tree cmd/upload_tree/main.go
