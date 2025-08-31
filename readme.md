swaggerapi/swagger-editor:latest
8080:8080

```
RUN go mod download
RUN go build -o app
```