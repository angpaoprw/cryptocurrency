sqlc:
	sqlc generate

mock:
	mockgen -source=db/sqlc/querier.go -package mockdb -destination db/mock/store.go

migrate:
	migrate create -ext sql -dir db/migration -seq init_schema

test:
	go test -v -cover ./...
swag:
	swag init