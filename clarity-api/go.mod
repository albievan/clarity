module github.com/albievan/clarity/clarity-api

go 1.22

require (
	github.com/go-chi/chi/v5 v5.0.14
	github.com/go-chi/cors v1.2.1
	github.com/go-sql-driver/mysql v1.7.1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/joho/godotenv v1.5.1
	github.com/microsoft/go-mssqldb v1.7.2
)

// Indirect dependencies pulled in by go-mssqldb.
// These are resolved when you run: go mod tidy
require (
	github.com/golang-sql/civil v0.0.0-20220223132316-b832511892a9 // indirect
	github.com/golang-sql/sqlexp v0.1.0 // indirect
	golang.org/x/crypto v0.27.0 // indirect
	golang.org/x/text v0.18.0 // indirect
)
