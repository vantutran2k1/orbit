package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

const CloudURL = "ws://localhost:8080/ws"

type TunnelMessage struct {
	JobID   string         `json:"job_id"`
	Payload map[string]any `json:"payload"`
}

var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

func main() {
	tenantID := flag.String("tenant", "", "your tenant uuid")
	targetPort := flag.String("port", "3000", "localhost port to forward to")
	flag.Parse()

	if *tenantID == "" {
		log.Fatal("please provide a tenant id: -tenant=uuid")
	}

	u, _ := url.Parse(CloudURL)
	q := u.Query()
	q.Set("tenant_id", *tenantID)
	u.RawQuery = q.Encode()

	log.Printf("connecting to orbit cloud: %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}

			var job TunnelMessage
			if err := json.Unmarshal(message, &job); err != nil {
				log.Println("bad json:", err)
				continue
			}

			go func(j TunnelMessage) {
				log.Printf("received job %s, forwarding...", j.JobID)
				forwardToLocalhost(j, *targetPort)
			}(job)
		}
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	log.Printf("orbit agent linked, forwarding [cloud] -> [localhost:%s]", *targetPort)

	select {
	case <-interrupt:
		log.Println("disconnecting...")
		c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		time.Sleep(time.Second)
	case <-done:
	}
}

func forwardToLocalhost(job TunnelMessage, port string) {
	localURL := fmt.Sprintf("http://localhost:%s/webhook", port)
	jsonBytes, _ := json.Marshal(job.Payload)

	resp, err := http.Post(localURL, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		log.Printf("failed to call localhost: %v", err)
		return
	}
	defer resp.Body.Close()

	log.Printf("localhost responded: %s", resp.Status)
}
