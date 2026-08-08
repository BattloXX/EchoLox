BINARY=EchoLox
VERSION=0.3.2
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

.PHONY: all arm64 armv7 amd64 zip clean tidy

all: tidy arm64 armv7 amd64

tidy:
	go mod tidy

arm64:
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY)-arm64 ./cmd/$(BINARY)

armv7:
	GOOS=linux GOARCH=arm GOARM=7 go build $(LDFLAGS) -o bin/$(BINARY)-armv7 ./cmd/$(BINARY)

amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY)-amd64 ./cmd/$(BINARY)

local:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/$(BINARY)

zip:
	zip -r $(BINARY)-$(VERSION).zip * \
		-x "*.git*" -x "bin/$(BINARY)" -x "*.zip" -x "*.tmp" -x "go.sum" -x "generated-image.png"

clean:
	rm -f bin/$(BINARY)-* bin/$(BINARY) $(BINARY)-*.zip
