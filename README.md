### Development

```bash
cd server
mkdir certs
openssl req -x509 -newkey rsa:4096 -sha256 -nodes -days 3650 \
    -keyout ./certs/pve-ssl.key \
    -out ./certs/pve-ssl.pem \
    -subj "/C=SK/ST=Slovakia/L=Bratislava/O=STU/OU=ARTEMIS/CN=localhost" \
    -addext "subjectAltName=DNS:localhost,DNS:*.localhost,IP:127.0.0.1,IP:::1"
```

```
MAX_CONCURRENT_REQUESTS=4
DATA_LOCATION="./data"
DATABASE_LOCATION="/opt/ludus/ludus.db"
LUDUS_ADMIN_URL=https://localhost:8081
LUDUS_URL=https://localhost:8080
PROXMOX_URL=https://localhost:8006
PROXMOX_CERT_PATH="./certs"                          
PROXMOX_NODE_NAME=raven
```


```
go mod download
go mod tidy
go run .
```

While developing I localforward ports from Ludus through ssh and set reading file limit to higher value

```bash
ulimit -n 10000
```

Change listen to 127.0.0.1 in `main.go` and comment out validateApiKey in `routes.go`

To preview the backend api you can use docker image `swaggerapi/swagger-editor:latest`