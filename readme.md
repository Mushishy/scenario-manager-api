### Development

```
cd server
go mod download
go mod tidy
go run .
```

### Documentation

You can edit `openapi.yaml` with
```
docker run -d -p 9000:8080 swaggerapi/swagger-editor:latest
```

### Deployment

Don't forget to check config.go first!
```
bash build.sh
bash deploy.sh
```