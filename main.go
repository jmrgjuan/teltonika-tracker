package main

import (
    "flag"
    "fmt"
    "log"
    "net"
    "time"
)

func main() {
    // --- Parámetros CLI ---
    serverIP := flag.String("server-ip", "127.0.0.1", "IP del servidor")
    serverPort := flag.Int("server-port", 5000, "Puerto del servidor")
    imei := flag.String("imei", "123456789012345", "IMEI del dispositivo simulado")

    retryConnect := flag.Int("retry-connect", 3, "Número de reintentos antes de pausa larga")
    sleepRetry := flag.Int("sleep-retry", 2, "Segundos entre reintentos")
    sleepNoConnect := flag.Int("sleep-noconnect", 30, "Segundos de espera si se agotan los reintentos")

    flag.Parse()

    log.Printf("[INFO] Simulador iniciado. IMEI=%s", *imei)

    address := fmt.Sprintf("%s:%d", *serverIP, *serverPort)

    for {
        log.Printf("[INFO] Intentando conectar a %s ...", address)

        var conn net.Conn
        var err error

        // --- Reintentos ---
        for attempt := 1; attempt <= *retryConnect; attempt++ {
            conn, err = net.Dial("tcp", address)
            if err == nil {
                log.Printf("[OK] Conectado al servidor en intento %d", attempt)
                break
            }

            log.Printf("[WARN] Fallo al conectar (intento %d/%d): %v",
                attempt, *retryConnect, err)

            time.Sleep(time.Duration(*sleepRetry) * time.Second)
        }

        // --- Si no se pudo conectar ---
        if err != nil {
            log.Printf("[ERROR] No se pudo conectar tras %d intentos. Esperando %d segundos...",
                *retryConnect, *sleepNoConnect)
            time.Sleep(time.Duration(*sleepNoConnect) * time.Second)
            continue
        }

        // --- Conexión establecida ---
        handleConnection(conn)

        // Si handleConnection termina, significa que la conexión se cerró
        log.Printf("[INFO] Conexión cerrada. Reintentando...")
    }
}

func handleConnection(conn net.Conn) {
    defer conn.Close()

    log.Printf("[INFO] Conexión activa con %s", conn.RemoteAddr())

    // Por ahora no enviamos nada. Más adelante enviaremos IMEI y codecs.
    // Dejamos la conexión abierta unos segundos para simular actividad.
    for {
        time.Sleep(5 * time.Second)
        log.Printf("[DEBUG] Conexión viva...")
    }
}
