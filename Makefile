all: test install

NOVENDOR = go list ./... | grep -v /vendor/

build:
	go build ./cmd/...

install:
	go install ./cmd/...

test:
	go test `${NOVENDOR}`
	#go run tests/tepleton/*.go

get_deps:
	go get -d ./...

update_deps:
	go get -d -u ./...

get_vendor_deps:
	go get github.com/Masterminds/glide
	glide install

build-docker:
	docker run -it --rm -v "$(PWD):/go/src/github.com/tepleton/basecoin" -w "/go/src/github.com/tepleton/basecoin" -e "CGO_ENABLED=0" golang:alpine go build ./cmd/basecoin
	docker build -t "tepleton/basecoin" .

clean:
	@rm -f ./basecoin

.PHONY: all build install test get_deps update_deps get_vendor_deps build-docker clean
