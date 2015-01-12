all: build test test-all

build:
	        go build -v

test:
	        go test -covermode=count -test.short -coverprofile=coverage.out -v

test-all:
	        go test -covermode=count -coverprofile=coverage.out -v
