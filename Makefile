GOOS := "linux"
ARTIFACT_PATH := bin
EXECUTE_FILE := goployer

format:
	go fmt ./...

local-build: format
	go build -o ${ARTIFACT_PATH}/local/${EXECUTE_FILE} main.go
	mv ${ARTIFACT_PATH}/local/${EXECUTE_FILE} /usr/local/bin
	rm -rf ${ARTIFACT_PATH}/local/${EXECUTE_FILE}

build: format local-build
	GOOS=${GOOS} go build -o ${ARTIFACT_PATH}/${EXECUTE_FILE} main.go

clean:
	@echo "  >  Cleaning build cache"
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean

