PROJECTNAME=$(shell basename "$(PWD)")

# Go related variables.
GOBASE=$(shell pwd)
GOPATH="$(GOBASE)/vendor:$(GOBASE):$(HOME)/go"
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

# Redirect error output to a file, so we can show it in development mode.
STDERR=/tmp/.$(PROJECTNAME)-stderr.txt

# PID file will keep the process id of the server
PID=/tmp/.$(PROJECTNAME).pid

# Make is verbose in Linux. Make it silent.
MAKEFLAGS += --silent

go-compile: go-clean go-cling go-examples

.PHONY: test
test: go-compile
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go test -v ./examples

go-examples:
	@echo "  >  building binaries..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -o $(GOBIN)/first examples/first.go

go-completer:
	@echo "  >  building completer..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBASE)/completer go build -o $(GOBIN)/completer build completer/completer.go

go-cling:
	@echo "  >  building dependencies..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -o $(GOBIN)/$(PROJECTNAME) $(GOFILES)

go-clean:
	@echo "  >  cleaning binaries..."
	rm -fr $(GOBIN)
