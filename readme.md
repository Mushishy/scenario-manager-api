### Development

```
cd server
go mod download
go mod tidy
source ../.env && go run .
```

### Documentation

You can edit `openapi.yaml` with
```
docker run -d -p 9000:8080 swaggerapi/swagger-editor:latest
```

### Deployment

Make sure that `docker-compose.yml` is placed in the same folder as folders `artemis-frontend` and `scenario-manager-api`.

```
docker-compose up -d
```