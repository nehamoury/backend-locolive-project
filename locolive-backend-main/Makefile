# Database URL from environment variable (never hardcode credentials)
# Usage: export DB_URL=postgresql://user:pass@host:5432/dbname || make migrateup
DB_URL ?= $(ENV_DB_URL)

migrateup:
	migrate -path db/migrations -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path db/migrations -database "$(DB_URL)" -verbose down

sqlc:
	sqlc generate

mock:
	~/go/bin/mockgen -package mockdb -destination internal/repository/mock/store.go privacy-social-backend/internal/repository Store

test:
	go test -v -cover ./...

server:
	go run cmd/server/main.go

redeploy:
	pm2 restart locolive-backend

.PHONY: migrateup migratedown sqlc mock test server redeploy
