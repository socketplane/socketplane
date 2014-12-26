build:
	docker build -t socketplane/socketplane .

test:
	go test -covermode=count -test.short -coverprofile=daemon.cover.out -coverpkg=./... ./daemon
	go test -covermode=count -test.short -coverprofile=datastore.cover.out -coverpkg=./... ./ipam
	go test -covermode=count -test.short -coverprofile=socketplane.cover.out

test-all:
	go test -covermode=count -coverprofile=daemon.cover.out -coverpkg=./... ./daemon
	go test -covermode=count -coverprofile=datastore.cover.out -coverpkg=./... ./ipam
	go test -covermode=count -coverprofile=socketplane.cover.out
