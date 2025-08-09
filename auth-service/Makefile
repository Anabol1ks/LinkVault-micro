doc:
	docker-compose -f docker-compose.yml up -d

run:
	go run cmd/main.go

proto:
	protoc api/proto/auth/v1/auth.proto \
		--proto_path=. \
		--proto_path=pkg \
		--go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		--validate_out=lang=go,paths=source_relative:gen

proto-install:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/envoyproxy/protoc-gen-validate@latest