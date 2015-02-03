.PHONY: build test test-all test-local test-all-local

build:
	docker build -t socketplane/socketplane .

coverage:
	gover
	goveralls -coverprofile=gover.coverprofile -service=$(CI_SERVICE) -repotoken=$(COVERALLS_TOKEN)

test:
	fig up -d
	docker run --privileged=true --net=host --rm -v $(shell pwd):/go/src/github.com/socketplane/socketplane -w /go/src/github.com/socketplane/socketplane davetucker/golang-ci:1.3 make test-local	
	fig stop

test-all:
	fig up -d
	docker run --privileged=true --net=host --rm -v $(shell pwd):/go/src/github.com/socketplane/socketplane -w /go/src/github.com/socketplane/socketplane davetucker/golang-ci:1.3 make test-all-local
	fig stop

test-local:
	go test -covermode=count -test.short -coverprofile=daemon.coverprofile -coverpkg=./... ./daemon
	go test -covermode=count -test.short -coverprofile=datastore.coverprofile -coverpkg=./... ./ipam
	go test -covermode=count -test.short -coverprofile=socketplane.coverprofile

test-all-local:
	go test -covermode=count -coverprofile=daemon.coverprofile -coverpkg=./... ./daemon
	go test -covermode=count -coverprofile=datastore.coverprofile -coverpkg=./... ./ipam
	go test -covermode=count -coverprofile=socketplane.coverprofile
