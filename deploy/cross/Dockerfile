FROM dockercore/golang-cross:1.13.10 as base

# The base image is not yet available for go 1.14.
# Let's just replace the Go that's installed with a newer one.
RUN rm -Rf /usr/local/go && mkdir /usr/local/go
RUN curl --fail --show-error --silent --location https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz \
    | tar xz --directory=/usr/local/go --strip-components=1

# Cross compile Skaffold for Linux, Windows and MacOS
ARG GOOS
ARG GOARCH
ARG TAGS
ARG LDFLAGS

WORKDIR /goployer
COPY . ./
RUN if [ "$GOOS" = "darwin" ]; then export CC=o64-clang CXX=o64-clang++; fi; \
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=1 \
    go build -tags "${TAGS}" -ldflags "${LDFLAGS}" -o /build/goployer main.go
