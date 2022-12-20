build: generate lint test
	@go build -o asit

lint:
	@go mod tidy
	@go fmt "./..." 
	@go vet "./..."

test:
	@go test "./..."

generate: install_protoc
	rm -rf pkg | true
	mkdir -p pkg/asit
	@protoc -I=. --go_out=pkg/asit proto/asit.proto

install_protoc:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.1

vendor:
	@go mod vendor

run_local: build
	@REDIS_ADDRS=localhost:6379 REVISION=localdev ./asit