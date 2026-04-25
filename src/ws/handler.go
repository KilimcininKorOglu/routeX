package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"routex/app"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	sendBuffer = 4
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  256,
	WriteBufferSize: 1024,
}

func StatsHandler(a app.Main, hub *Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Warn().Err(err).Msg("WebSocket upgrade failed")
			return
		}

		client := &Client{
			conn: conn,
			send: make(chan []byte, sendBuffer),
		}

		if !hub.Register(client) {
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "too many clients"))
			conn.Close()
			return
		}

		go writePump(client, hub)
		go readPump(client, hub)
		go statsPump(r.Context(), client, a, hub)
	}
}

func readPump(client *Client, hub *Hub) {
	defer func() {
		hub.Unregister(client)
		client.conn.Close()
	}()

	client.conn.SetReadLimit(256)
	client.conn.SetReadDeadline(time.Now().Add(pongWait))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := client.conn.ReadMessage(); err != nil {
			break
		}
	}
}

func writePump(client *Client, hub *Hub) {
	pingTicker := time.NewTicker(pingPeriod)
	defer func() {
		pingTicker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := client.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-pingTicker.C:
			client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func statsPump(ctx context.Context, client *Client, a app.Main, hub *Hub) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			snap := a.GetStats()
			data, err := json.Marshal(snap)
			if err != nil {
				continue
			}
			select {
			case client.send <- data:
			default:
			}
		case <-ctx.Done():
			return
		}
	}
}
