LOCAL_BIN:=$(CURDIR)/bin

install-deps:
	GOBIN=$(LOCAL_BIN) go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	GOBIN=$(LOCAL_BIN) go install -mod=mod google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	GOBIN=$(LOCAL_BIN) go install github.com/jmattheis/goverter/cmd/goverter@v1.8.1
	GOBIN=$(LOCAL_BIN) go install github.com/magefile/mage@latest

get-deps:
	go get -u google.golang.org/protobuf/cmd/protoc-gen-go
	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc


generate:
	make generate-note-api

generate-orders-api:
	mkdir -p internal/pb/v1
	protoc --proto_path api/v1 \
	--go_out=./internal/pb/v1 --experimental_allow_proto3_optional --go_opt=paths=source_relative \
	--plugin=protoc-gen-go=bin/protoc-gen-go \
	--go-grpc_out=internal/pb/v1 --go-grpc_opt=paths=source_relative \
	--plugin=protoc-gen-go-grpc=bin/protoc-gen-go-grpc \
	api/v1/orders.proto

generate-common-api:
	mkdir -p internal/pb/v1
	protoc --proto_path api/v1 \
	--go_out=./internal/pb/v1 --experimental_allow_proto3_optional --go_opt=paths=source_relative \
	--plugin=protoc-gen-go=bin/protoc-gen-go \
	--go-grpc_out=internal/pb/v1 --go-grpc_opt=paths=source_relative \
	--plugin=protoc-gen-go-grpc=bin/protoc-gen-go-grpc \
	api/v1/common.proto

generate-instruments-api:
	mkdir -p internal/pb/v1
	protoc --proto_path api/v1 \
	--go_out=./internal/pb/v1 --experimental_allow_proto3_optional --go_opt=paths=source_relative \
	--plugin=protoc-gen-go=bin/protoc-gen-go \
	--go-grpc_out=internal/pb/v1 --go-grpc_opt=paths=source_relative \
	--plugin=protoc-gen-go-grpc=bin/protoc-gen-go-grpc \
	api/v1/instruments.proto

generate-market-data-api:
	mkdir -p internal/pb/v1
	protoc --proto_path api/v1 \
	--go_out=./internal/pb/v1 --experimental_allow_proto3_optional --go_opt=paths=source_relative \
	--plugin=protoc-gen-go=bin/protoc-gen-go \
	--go-grpc_out=internal/pb/v1 --go-grpc_opt=paths=source_relative \
	--plugin=protoc-gen-go-grpc=bin/protoc-gen-go-grpc \
	api/v1/marketdata.proto

wire-generate:
	$(LOCAL_BIN)/goverter .