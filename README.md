## GoRoadmap (car plate tracking)

Small Go REST API for tracking vehicles by plate and generating simple reports.

### Requirements

- Go (any recent 1.x)
- Postgres (local via Docker is fine)

### Environment variables

- **`DATABASE_URL`**: Postgres DSN, for example:
    - `postgres://postgres:postgres@localhost:5432/goroadmap?sslmode=disable`
    - `"postgres://postgres:postgres@localhost:5432/goroadmap_test?sslmode=disable"` (test env)
- **`PORT`**: optional port number (default: `8080`)
- **`ADDR`**: optional full bind address (overrides `PORT`), e.g. `:8080` or `127.0.0.1:8080`
- **`FIXTURE`**: optional, if set to `1` enables `/fixture` endpoint (dev-only)
- **`API_KEY`**: optional, if auth is enabled for `/fixture` (see `main.go`)

### Database setup

- **Create Database**
  ```powershell
  docker exec -i local-postgres psql -U postgres -d postgres -c "CREATE DATABASE goroadmap;"
  ```
- **Apply migration**
  ```powershell
  Get-Content .\migrations\001_init.up.sql | docker exec -i local-postgres psql -U postgres -d goroadmap -f -
  ```
- **Create test Database**
  ```powershell
  docker exec -i local-postgres psql -U postgres -d postgres -c "CREATE DATABASE goroadmap_test;"
  ```
- **Apply migration to test Database**
  ```powershell
  Get-Content .\migrations\001_init.up.sql | docker exec -i local-postgres psql -U postgres -d goroadmap_test -f -
  ```

### Run

- **Prod**
  ```powershell
  $env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/goroadmap?sslmode=disable"
  go run .
  ```

- **Test**
  ```powershell
  $env:DATABASE_URL = "postgres://postgres:postgres@localhost:5432/goroadmap_test?sslmode=disable"
  go test ./...
  ```

### API quickstart

- **Health**

```powershell
curl.exe -i "http://localhost:8080/health"
```

- **Create track event**

```powershell
curl.exe -i -X POST "http://localhost:8080/tracks" -H "Content-Type: application/json" -d "{\"plate\":\"AB12CDE\",\"note\":\"near gate\"}"
```

- **List track events**

```powershell
curl.exe -i "http://localhost:8080/tracks"
curl.exe -i "http://localhost:8080/tracks?plate=AB&limit=20&offset=0"
```

- **Get by id**

```powershell
curl.exe -i "http://localhost:8080/tracks/1"
```

- **Delete by id**

```powershell
curl.exe -i -X DELETE "http://localhost:8080/tracks/1"
```

- **Report**

```powershell
curl.exe -i "http://localhost:8080/report"
curl.exe -i "http://localhost:8080/report?plate=AB"
curl.exe -i "http://localhost:8080/report?from=2026-04-01T00:00:00Z&to=2026-05-01T00:00:00Z"
curl.exe -i "http://localhost:8080/report?from=2026-04-01&to=2026-05-01"
```