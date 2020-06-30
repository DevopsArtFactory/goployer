GOOS := "linux"
ARTIFACT_PATH := bin
EXECUTE_FILE := goployer

format:
	go fmt ./...

build: format
	GOOS=${GOOS} go build -o ${ARTIFACT_PATH}/${EXECUTE_FILE} main.go

clean:
	@echo "  >  Cleaning build cache"
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean

