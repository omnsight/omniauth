# Omniauth

## Run Locally

### With Docker Compose (Recommended)

```bash
docker-compose up --build
```

This will start both the Keycloak instance and the omniauth service. Keycloak will be available at http://localhost:8080 and the service at http://localhost:8081.

### Dependencies

To upgrade internal dependencies:

```bash
go clean -cache -modcache
GOPROXY=direct go get github.com/omnsight/omniscent-library@<branch>
```

Buf build:

```bash
buf registry login buf.build

buf dep update

buf format -w
buf lint

buf generate
buf push

go mod tidy
```

### Testing

Run unit tests. You can view arangodb dashboard at http://localhost:8529.

```bash
go test -v ./... -run <test name>
```