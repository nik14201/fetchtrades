PROJECTNAME="main"
# Go related variables.
GOBASE=$(bash pwd)
GOPATH="$(GOBASE)/vendor:$(GOBASE)"
GOBIN=$(GOBASE)
GOFILES="main.go"
GOTEST="calc/testmain.go"

build:
	@echo "  >  Building binary..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go build -o main -mod vendor $(GOFILES)

push:
	@echo "  >  Push docker..."
	./build.sh

clean:
	@echo "  >  Cleaning build cache"
	rm main

test:
	@echo "  >  Test..."
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go run $(GOTEST)