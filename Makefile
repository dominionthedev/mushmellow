.PHONY: build test vet fmt validate clean

build:
	go build -o bin/mushmellow .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

validate: build
	cd examples/basic && ../../bin/mushmellow validate

clean:
	rm -rf bin
