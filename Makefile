.PHONY: all build test clean benchmark studio

all: build test

build: build-web build-go

build-web:
	cd web && npm install && npm run build

build-go:
	go build -o codekernel.exe ./cmd/codekernel

test:
	go test -v ./internal/...

benchmark: build-go
	./codekernel.exe benchmark

clean:
	rm -rf codekernel.exe bin/ dist/ web/dist .codekernel/
