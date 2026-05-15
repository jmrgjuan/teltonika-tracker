# teltonika-tracker
Simula un tracker Teltonika y mantiene una conexión TCP al servidor configurado.

## Iniciar el proyecto Go

Este repositorio contiene un ejecutable Go que intenta conectarse a un servidor TCP y mantiene la conexión abierta.

### Comandos útiles

- `go run main.go` — ejecutar directamente el programa
- `go build -o bin/teltonika-tracker main.go` — compilar el ejecutable
- `./bin/teltonika-tracker` — ejecutar el binario compilado

### Opciones disponibles

- `-server-ip` — IP del servidor (por defecto `127.0.0.1`)
- `-server-port` — puerto del servidor (por defecto `5000`)
- `-imei` — IMEI del dispositivo simulado (por defecto `123456789012345`)
- `-retry-connect` — número de reintentos antes de parar y esperar más tiempo (por defecto `3`)
- `-sleep-retry` — segundos entre reintentos de conexión (por defecto `2`)
- `-sleep-noconnect` — segundos de espera si se agotan los reintentos (por defecto `30`)

## Ejemplos de ejecución

Ejecutar con valores por defecto:

```bash
go run main.go
```

Ejecutar contra un servidor local en el puerto 5000 con IMEI personalizado:

```bash
go run main.go \
  -server-ip 127.0.0.1 \
  -server-port 5000 \
  -imei 123456789012345
```

Compilar y ejecutar el binario:

```bash
go build -o bin/teltonika-tracker main.go
./bin/teltonika-tracker -server-ip 192.168.1.10 -server-port 5000 -imei 123456789012345
```

Aumentar los reintentos y reducir el intervalo entre ellos:

```bash
go run main.go \
  -retry-connect 5 \
  -sleep-retry 1
```
