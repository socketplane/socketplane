.PHONY: build coverage test test-all test-local test-all-local

build:
	docker build -t socketplane/socketplane .

coverage:
	sh tools/combine-coverage.sh
	goveralls -coverprofile=socketplane.coverprofile -service=$(CI_SERVICE) -repotoken=$(COVERALLS_TOKEN)

test:
	docker-compose up -d ovs
	docker run --cap-add=NET_ADMIN --cap-add SYS_ADMIN --net=host --rm -v $(shell pwd):/go/src/github.com/socketplane/socketplane -w /go/src/github.com/socketplane/socketplane davetucker/golang-ci:1.3 make test-local	
	docker-compose stop

test-all:
	docker-compose up -d ovs
	docker run --cap-add=NET_ADMIN --cap-add SYS_ADMIN --net=host --rm -v $(shell pwd):/go/src/github.com/socketplane/socketplane -w /go/src/github.com/socketplane/socketplane davetucker/golang-ci:1.3 make test-all-local
	docker-compose stop

test-local:
	go test -covermode=count -test.short -coverprofile=daemon.cover.out -coverpkg=./... ./daemon
	go test -covermode=count -test.short -coverprofile=socketplane.cover.out

test-all-local:
	go test -covermode=count -coverprofile=daemon.cover.out -coverpkg=./... ./daemon
	go test -covermode=count -coverprofile=socketplane.cover.out
