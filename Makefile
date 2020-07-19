GOOS := "linux"
ARTIFACT_PATH := bin
EXECUTE_FILE := goployer

format:
	go fmt ./...

local-build: format
	go build -o ${ARTIFACT_PATH}/local/${EXECUTE_FILE} main.go
	mv ${ARTIFACT_PATH}/local/${EXECUTE_FILE} /usr/local/bin
	rm -rf ${ARTIFACT_PATH}/local

build: format local-build
	GOOS=${GOOS} go build -o ${ARTIFACT_PATH}/${EXECUTE_FILE} main.go

commit: format local-build
	rm -rf ${ARTIFACT_PATH}
	git add .
	git commit
	git push origin $(shell git branch --show-current)


clean:
	@echo "  >  Cleaning build cache"
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) go clean

