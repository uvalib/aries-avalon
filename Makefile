GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test

build: darwin

all: darwin linux 

darwin:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -a -o bin/aries-virgo.darwin *.go

linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -a -installsuffix cgo -o bin/aries-virgo.linux *.go

clean:
	$(GOCLEAN)
	rm -rf bin
