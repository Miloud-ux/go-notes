package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"time"
)

func main() {

	backends := []string{
		"http://localhost:8000",
		"http://localhost:8001",
	}

	proxies := make([]*httputil.ReverseProxy, len(backends))

	for idx, backend := range backends {
		u, err := url.Parse(backend)
		if err != nil {
			log.Fatalf("URL pasring error: %v", err)
		}

		proxies[idx] = httputil.NewSingleHostReverseProxy(u)
	}

	var mu sync.Mutex
	idx := 0

	requestRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		p := proxies[idx]
		idx = (idx + 1) % len(proxies)
		mu.Unlock()

		p.ServeHTTP(w, r)
	})

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      requestRouter,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Println("Reverse Listening on addr :8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Proxy ListenAndServe() err: %s\n", err)
		}
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	<-stopChan
	log.Printf("Interrupted! Shutting down reverse proxy ")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv.Shutdown(ctx)
	log.Println("reverse proxy gracefully stopped")

}
