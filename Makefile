PACKAGES=$(shell go list ./... | grep -v '/vendor/')
PACKAGES_NOCLITEST=$(shell go list ./... | grep -v '/vendor/' | grep -v github.com/tepleton/tepleton-sdk/cmd/ton/cli_test)
COMMIT_HASH := $(shell git rev-parse --short HEAD)
BUILD_FLAGS = -tags netgo -ldflags "-X github.com/tepleton/tepleton-sdk/version.GitCommit=${COMMIT_HASH}"

all: get_tools get_vendor_deps install install_examples test_lint test

########################################
### CI

ci: get_tools get_vendor_deps install test_cover test_lint test

########################################
### Build

# This can be unified later, here for easy demos
build:
ifeq ($(OS),Windows_NT)
	go build $(BUILD_FLAGS) -o build/tond.exe ./cmd/ton/cmd/tond
	go build $(BUILD_FLAGS) -o build/toncli.exe ./cmd/ton/cmd/toncli
else
	go build $(BUILD_FLAGS) -o build/tond ./cmd/ton/cmd/tond
	go build $(BUILD_FLAGS) -o build/toncli ./cmd/ton/cmd/toncli
endif

build_examples:
ifeq ($(OS),Windows_NT)
	go build $(BUILD_FLAGS) -o build/basecoind.exe ./examples/basecoin/cmd/basecoind
	go build $(BUILD_FLAGS) -o build/basecli.exe ./examples/basecoin/cmd/basecli
	go build $(BUILD_FLAGS) -o build/democoind.exe ./examples/democoin/cmd/democoind
	go build $(BUILD_FLAGS) -o build/democli.exe ./examples/democoin/cmd/democli
else
	go build $(BUILD_FLAGS) -o build/basecoind ./examples/basecoin/cmd/basecoind
	go build $(BUILD_FLAGS) -o build/basecli ./examples/basecoin/cmd/basecli
	go build $(BUILD_FLAGS) -o build/democoind ./examples/democoin/cmd/democoind
	go build $(BUILD_FLAGS) -o build/democli ./examples/democoin/cmd/democli
endif

install:
	go install $(BUILD_FLAGS) ./cmd/ton/cmd/tond
	go install $(BUILD_FLAGS) ./cmd/ton/cmd/toncli

install_examples:
	go install $(BUILD_FLAGS) ./examples/basecoin/cmd/basecoind
	go install $(BUILD_FLAGS) ./examples/basecoin/cmd/basecli
	go install $(BUILD_FLAGS) ./examples/democoin/cmd/democoind
	go install $(BUILD_FLAGS) ./examples/democoin/cmd/democli

install_debug:
	go install $(BUILD_FLAGS) ./cmd/ton/cmd/tondebug

dist:
	@bash publish/dist.sh
	@bash publish/publish.sh

########################################
### Tools & dependencies

check_tools:
	cd tools && $(MAKE) check_tools

update_tools:
	cd tools && $(MAKE) update_tools

get_tools:
	cd tools && $(MAKE) get_tools

get_vendor_deps:
	@rm -rf vendor/
	@echo "--> Running dep ensure"
	@dep ensure -v

draw_deps:
	@# requires brew install graphviz or apt-get install graphviz
	go get github.com/RobotsAndPencils/goviz
	@goviz -i github.com/tepleton/tepleton-sdk/cmd/ton/cmd/tond -d 2 | dot -Tpng -o dependency-graph.png


########################################
### Documentation

godocs:
	@echo "--> Wait a few seconds and visit http://localhost:6060/pkg/github.com/tepleton/tepleton-sdk/types"
	godoc -http=:6060


########################################
### Testing

test: test_unit

test_cli:
	@go test -count 1 -p 1 `go list github.com/tepleton/tepleton-sdk/cmd/ton/cli_test`

test_unit:
	@go test $(PACKAGES_NOCLITEST)

test_race:
	@go test -race $(PACKAGES_NOCLITEST)

test_cover:
	@bash tests/test_cover.sh

test_lint:
	gometalinter.v2 --disable-all --enable='golint' --enable='misspell' --enable='unparam' --enable='unconvert' --enable='ineffassign' --linter='vet:go vet -composites=false:PATH:LINE:MESSAGE' --enable='vet' --deadline=500s --vendor ./...
	!(gometalinter.v2 --disable-all --enable='errcheck' --vendor ./... | grep -v "client/")
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" | xargs gofmt -d -s

format:
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" | xargs gofmt -w -s
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" | xargs misspell -w

benchmark:
	@go test -bench=. $(PACKAGES_NOCLITEST)


########################################
### Devdoc

DEVDOC_SAVE = docker commit `docker ps -a -n 1 -q` devdoc:local

devdoc_init:
	docker run -it -v "$(CURDIR):/go/src/github.com/tepleton/tepleton-sdk" -w "/go/src/github.com/tepleton/tepleton-sdk" tepleton/devdoc echo
	# TODO make this safer
	$(call DEVDOC_SAVE)

devdoc:
	docker run -it -v "$(CURDIR):/go/src/github.com/tepleton/tepleton-sdk" -w "/go/src/github.com/tepleton/tepleton-sdk" devdoc:local bash

devdoc_save:
	# TODO make this safer
	$(call DEVDOC_SAVE)

devdoc_clean:
	docker rmi -f $$(docker images -f "dangling=true" -q)

devdoc_update:
	docker pull tepleton/devdoc


########################################
### Local validator nodes using docker and docker-compose

# Build linux binary
build-linux:
	GOOS=linux GOARCH=amd64 $(MAKE) build

build-docker-tondnode:
	$(MAKE) -C networks/local

# Run a 4-node testnet locally
localnet-start: localnet-stop
	@if ! [ -f build/node0/tond/config/genesis.json ]; then docker run --rm -v $(CURDIR)/build:/tond:Z tepleton/tondnode testnet --v 4 --o . --starting-ip-address 192.168.10.2 ; fi
	docker-compose up

# Stop testnet
localnet-stop:
	docker-compose down

########################################
### Remote validator nodes using terraform and ansible

TESTNET_NAME?=remotenet
SERVERS?=4
BINARY=$(CURDIR)/build/tond
remotenet-start:
	@if [ -z "$(DO_API_TOKEN)" ]; then echo "DO_API_TOKEN environment variable not set." ; false ; fi
	@if ! [ -f $(HOME)/.ssh/id_rsa.pub ]; then ssh-keygen ; fi
	@if [ -z "`file $(BINARY) | grep 'ELF 64-bit'`" ]; then echo "Please build a linux binary using 'make build-linux'." ; false ; fi
	cd networks/remote/terraform && terraform init && terraform apply -var DO_API_TOKEN="$(DO_API_TOKEN)" -var SSH_PUBLIC_FILE="$(HOME)/.ssh/id_rsa.pub" -var SSH_PRIVATE_FILE="$(HOME)/.ssh/id_rsa" -var TESTNET_NAME="$(TESTNET_NAME)" -var SERVERS="$(SERVERS)"
	cd networks/remote/ansible && ANSIBLE_HOST_KEY_CHECKING=False ansible-playbook -i inventory/digital_ocean.py -l "$(TESTNET_NAME)" -e BINARY=$(BINARY) -e TESTNET_NAME="$(TESTNET_NAME)" setup-validators.yml
	cd networks/remote/ansible && ansible-playbook -i inventory/digital_ocean.py -l "$(TESTNET_NAME)" start.yml

remotenet-stop:
	@if [ -z "$(DO_API_TOKEN)" ]; then echo "DO_API_TOKEN environment variable not set." ; false ; fi
	cd networks/remote/terraform && terraform destroy -var DO_API_TOKEN="$(DO_API_TOKEN)" -var SSH_PUBLIC_FILE="$(HOME)/.ssh/id_rsa.pub" -var SSH_PRIVATE_FILE="$(HOME)/.ssh/id_rsa"

remotenet-status:
	cd networks/remote/ansible && ansible-playbook -i inventory/digital_ocean.py -l "$(TESTNET_NAME)" status.yml

# To avoid unintended conflicts with file names, always add to .PHONY
# unless there is a reason not to.
# https://www.gnu.org/software/make/manual/html_node/Phony-Targets.html
.PHONY: build build_examples install install_examples install_debug dist check_tools get_tools get_vendor_deps draw_deps test test_cli test_unit test_cover test_lint benchmark devdoc_init devdoc devdoc_save devdoc_update build-linux build-docker-tondnode localnet-start localnet-stop remotenet-start remotenet-stop remotenet-status format
