## Database migrations

Run the custom migration helper to apply any `.sql` scripts inside the `migrations` directory:

```bash
go run ./cmd/migrate -path migrations -database "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" up
```

If the `-database` flag is omitted, the helper falls back to the connection information derived from the usual environment variables (`DB_HOST`, `DB_PORT`, etc.).
