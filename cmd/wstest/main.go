// cmd/wstest — ручная проверка WS-канала: подключается, читает N сообщений,
// печатает в stdout, выходит. Нужен только для dev-верификации flow этапа 9.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/coder/websocket"
)

func main() {
	sid := flag.String("session", "", "session id")
	jwt := flag.String("jwt", "", "access JWT")
	n := flag.Int("n", 2, "messages to read")
	flag.Parse()
	if *sid == "" || *jwt == "" {
		log.Fatal("--session and --jwt required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	url := "ws://localhost:8080/ws/sessions/" + *sid + "/teacher"
	proto := "bearer." + *jwt
	conn, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
		Subprotocols: []string{proto},
	})
	if err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer conn.CloseNow()

	for i := 0; i < *n; i++ {
		_, data, err := conn.Read(ctx)
		if err != nil {
			log.Fatalf("read[%d]: %v", i, err)
		}
		fmt.Fprintf(os.Stdout, "MSG[%d] %s\n", i, string(data))
	}
	_ = conn.Close(websocket.StatusNormalClosure, "done")
}
