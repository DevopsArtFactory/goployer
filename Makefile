GOOS := "linux"
ARTIFACT_PATH := bin
EXECUTE_FILE := deployer

build:
	GOOS=${GOOS} go build -o ${ARTIFACT_PATH}/${EXECUTE_FILE} main.go

clean:
	@echo "  >  Cleaning build cache"
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean

