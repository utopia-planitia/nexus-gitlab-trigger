package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	secretPath := flag.String("secret", "./secret", "secret used to sign the payload form nexus")
	listen := flag.String("listen", ":8080", "address to serve on, ':8080'")
	gitlabServer := flag.String("gitlab", "http://gitlab.gitlab.svc", "url of gitlab server")
	projectsPath := flag.String("projects", "./projects.yaml", "path to list of projects")

	flag.Parse()

	log.Printf("secret: %s\n", *secretPath)
	log.Printf("listen: %s\n", *listen)
	log.Printf("gitlab: %s\n", *gitlabServer)
	log.Printf("projects: %s\n", *projectsPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/nexus", func(w http.ResponseWriter, r *http.Request) {
		nexusHandler(w, r, *secretPath, gitlab{serverURL: *gitlabServer, projectsConfig: *projectsPath})
	})

	server := &http.Server{
		Addr:         *listen,
		Handler:      mux,
		ReadTimeout:  1800 * time.Second,
		WriteTimeout: 1800 * time.Second,
	}

	// wait for an exit signal
	stop := make(chan os.Signal, 2)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		err := server.Shutdown(context.Background())
		if err != nil {
			log.Fatalf("server shutdown failed: %s\n", err)
		}
	}()

	// serve requests
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
