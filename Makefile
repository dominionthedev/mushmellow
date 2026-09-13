.PHONY: build test vet fmt validate install clean

build:
	go build -o bin/mushmellow .

install: build
	cp bin/mushmellow /usr/local/bin/mushmellow

test:
	go test ./... -race

vet:
	go vet ./...

fmt:
	gofmt -w .

validate: build
	cd examples/basic && ../../bin/mushmellow validate

clean:
	rm -rf bin
