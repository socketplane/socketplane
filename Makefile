.PHONY: build coverage test test-all test-local test-all-local

build:
	docker build -t socketplane/socketplane .

coverage:
	sh tools/combine-coverage.sh
	goveralls -coverprofile=socketplane.coverprofile -service=$(CI_SERVICE) -repotoken=$(COVERALLS_TOKEN)

test:
	fig up -d
	docker run --cap-add=NET_ADMIN --net=container:socketplane_ovs_1 --rm -v $(shell pwd):/go/src/github.com/socketplane/socketplane -w /go/src/github.com/socketplane/socketplane davetucker/golang-ci:1.3 make test-local	
	fig stop

test-all:
	fig up -d
	docker run --cap-add=NET_ADMIN --net=container:socketplane_ovs_1 --rm -v $(shell pwd):/go/src/github.com/socketplane/socketplane -w /go/src/github.com/socketplane/socketplane davetucker/golang-ci:1.3 make test-all-local
	fig stop

test-local:
	go test -covermode=count -test.short -coverprofile=daemon.cover.out -coverpkg=./... ./daemon
	go test -covermode=count -test.short -coverprofile=datastore.cover.out -coverpkg=./... ./ipam
	go test -covermode=count -test.short -coverprofile=socketplane.cover.out

test-all-local:
	go test -covermode=count -coverprofile=daemon.cover.out -coverpkg=./... ./daemon
	go test -covermode=count -coverprofile=datastore.cover.out -coverpkg=./... ./ipam
	go test -covermode=count -coverprofile=socketplane.cover.out
