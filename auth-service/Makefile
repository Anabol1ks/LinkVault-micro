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