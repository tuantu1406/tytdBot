package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tytd_go/internal/eventinspector"
)

func main() {
	addr := flag.String("edge-cdp", "127.0.0.1:9222", "Edge remote-debugging address")
	out := flag.String("out", "event-inspector.json", "output JSON path")
	duration := flag.Duration("duration", 0, "stop after duration (0 means Ctrl+C)")
	flag.Parse()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if *duration > 0 {
		var c context.CancelFunc
		ctx, c = context.WithTimeout(ctx, *duration)
		defer c()
	}
	t := &eventinspector.Timeline{}
	fmt.Printf("Event Inspector: read-only CDP observation on %s\n", *addr)
	err := eventinspector.Observe(ctx, *addr, t)
	if err != nil && ctx.Err() == nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	b, _ := json.MarshalIndent(t, "", "  ")
	if err := os.WriteFile(*out, b, 0600); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("saved %d frames to %s at %s\n", len(t.Frames), *out, time.Now().Format(time.RFC3339))
}
