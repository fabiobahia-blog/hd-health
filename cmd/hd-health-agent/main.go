package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hd-health/hd-health/internal/platform"
	"github.com/hd-health/hd-health/internal/report"
	"github.com/hd-health/hd-health/internal/store"
)

func main() {
	interval := flag.Duration("interval", time.Hour, "snapshot interval")
	once := flag.Bool("once", false, "run one snapshot and exit")
	dbPath := flag.String("db", "", "database path")
	flag.Parse()

	st, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	plat := platform.Current()
	run := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		b := &report.Builder{Platform: plat, Store: st}
		r, code, err := b.Build(ctx)
		if err != nil {
			log.Printf("snapshot error: %v", err)
			return
		}
		log.Printf("snapshot ok hostname=%s volumes=%d findings=%d exit=%d",
			r.Hostname, len(r.Volumes), len(r.Findings), code)
	}

	if *once {
		run()
		os.Exit(0)
	}

	run()
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-ticker.C:
			run()
		case s := <-sig:
			log.Printf("shutting down: %v", s)
			return
		}
	}
}
