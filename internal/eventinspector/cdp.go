package eventinspector

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"strings"
	"time"
)

type cdpTarget struct {
	Type                 string `json:"type"`
	Title                string `json:"title"`
	URL                  string `json:"url"`
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}
type cdpMessage struct {
	Method string `json:"method"`
	Params struct {
		Response struct {
			URL         string `json:"url"`
			PayloadData string `json:"payloadData"`
		} `json:"response"`
		Request struct {
			URL string `json:"url"`
		} `json:"request"`
		Timestamp float64 `json:"timestamp"`
	} `json:"params"`
}

// Observe attaches to an existing Edge DevTools target and records only frames.
// It does not call Runtime.evaluate, fetch, or any game endpoint.
func Observe(ctx context.Context, addr string, t *Timeline) error {
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/json/list", nil)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("CDP target list: %w", err)
	}
	defer resp.Body.Close()
	var targets []cdpTarget
	if err = json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return err
	}
	var target *cdpTarget
	for i := range targets {
		candidate := &targets[i]
		if candidate.WebSocketDebuggerURL == "" || candidate.URL == "" || strings.Contains(strings.ToLower(candidate.Title), "devtools") {
			continue
		}
		isGame := strings.Contains(candidate.URL, "h5.slfuntap.com") || strings.Contains(candidate.URL, "slfuntap.com") || strings.Contains(candidate.URL, "cdn-xddq-vn.hwrescdn.com")
		if target == nil && candidate.Type == "page" {
			target = candidate
		}
		if isGame {
			target = candidate
			if strings.Contains(candidate.URL, "cdn-xddq-vn.hwrescdn.com") {
				break
			}
		}
	}
	if target == nil {
		return fmt.Errorf("no Edge page target found at %s", addr)
	}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, target.WebSocketDebuggerURL, nil)
	if err != nil {
		return fmt.Errorf("CDP websocket: %w", err)
	}
	defer conn.Close()
	readDone := make(chan struct{})
	defer close(readDone)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-readDone:
		}
	}()
	for i, method := range []string{"Network.enable", "Page.enable"} {
		b, _ := json.Marshal(map[string]any{"id": i + 1, "method": method})
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			return err
		}
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, b, err := conn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return err
		}
		var m cdpMessage
		if json.Unmarshal(b, &m) != nil {
			continue
		}
		dir, url := "", ""
		switch m.Method {
		case "Network.webSocketFrameReceived":
			dir = "received"
			url = m.Params.Response.URL
		case "Network.webSocketFrameSent":
			dir = "sent"
			url = m.Params.Request.URL
		default:
			continue
		}
		AddFrame(t, Frame{Timestamp: time.Now().UTC(), Direction: dir, URL: url, Payload: m.Params.Response.PayloadData})
		if decoded, decodeErr := DecodeProto([]byte(m.Params.Response.PayloadData)); decodeErr == nil {
			if raw, ok := decoded["activityType"].(string); ok && raw != "" {
				known := false
				for _, a := range t.Activities {
					known = known || a.Type == raw
				}
				t.Activities = append(t.Activities, Activity{Timestamp: time.Now().UTC(), Type: raw, FrameIndex: len(t.Frames) - 1, New: !known})
			}
		}
	}
}
