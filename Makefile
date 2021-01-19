GOOS = $(shell go env GOOS)

server:
	go build -o bin/foodtruck-server -a -ldflags '-extldflags "-static"' ./cmd/foodtruck-server

client-all: client-linux client-windows client-darwin client-solaris client-aix

client-%: OS = $(subst client-,,$@)
client-%: ARCH = amd64
client-aix: ARCH = ppc64

client-linux client-windows client-darwin client-solaris client-aix:
	CGO_ENABLED=0 GOOS=${OS} GOARCH=${ARCH} go build -o bin/foodtruck-client-${OS}-${ARCH} -a -ldflags '-extldflags "-static"' ./cmd/foodtruck-client

.PHONY: server client-linux client-windows client-darwin client-solaris client-aix 
