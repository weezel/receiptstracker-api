# CGO_ENABLED=0 == static by default
GO		= go
# -s removes symbol table and -ldflags -w debugging symbols
LDFLAGS		= -trimpath -ldflags "-s -w"
GOOS		= linux
GOARCH		= amd64
BINARY		= receiptstracker-api

.PHONY: all analysis obsd test

# Defaults to Linux
linux:
	CGO_ENABLED=0 $(GO) build $(LDFLAGS) -o $(BINARY)
debug:
	CGO_ENABLED=1 $(GO) build $(LDFLAGS) -o $(BINARY)
obsd:
	GOOS=openbsd $(GO) build $(LDFLAGS) -o $(BINARY)_obsd
test:
	go clean -testcache && go test ./...
