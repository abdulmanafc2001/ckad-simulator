package main

import (
	"log"
	"os"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/checker"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/router"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/memory"
)

func main() {
	// Seed the in-memory store with the CKAD question bank.
	repo := store.NewRepository(memory.New(seedQuestions()))

	// The checker executes kubectl against the current kubeconfig context
	// (minikube) to prepare task environments and grade answers.
	chk := checker.New()
	if b := os.Getenv("KUBECTL_BIN"); b != "" {
		chk.Binary = b
	}

	svc := store.NewService(repo, chk)

	r := router.New(svc)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("CKAD Simulator API listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
