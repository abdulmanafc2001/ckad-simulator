package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/checker"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/models"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/router"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/memory"
	"github.com/abdulmanafc2001/ckad-simulator/backend/internal/store/sqlite"
)

func main() {
	repo := store.NewRepository(openRepository(seedQuestions()))

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

// openRepository picks the session store. Exams are persisted to SQLite by
// default so an in-progress exam survives a restart and finished ones stay
// reviewable; CKAD_DB points the database elsewhere, and CKAD_DB=off keeps
// everything in memory.
//
// A database that cannot be opened degrades to memory rather than stopping
// the server: losing exam history is an inconvenience, but refusing to boot
// over it would cost the candidate their practice session.
func openRepository(questions []*models.Question) store.Repository {
	path := os.Getenv("CKAD_DB")
	if path == "off" {
		log.Print("session store: in-memory (CKAD_DB=off) — exams will not survive a restart")
		return memory.New(questions)
	}
	if path == "" {
		var err error
		if path, err = defaultDBPath(); err != nil {
			log.Printf("session store: falling back to memory (%v)", err)
			return memory.New(questions)
		}
	}

	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			log.Printf("session store: falling back to memory (%v)", err)
			return memory.New(questions)
		}
	}

	db, err := sqlite.New(path, questions)
	if err != nil {
		log.Printf("session store: falling back to memory (%v)", err)
		return memory.New(questions)
	}
	log.Printf("session store: %s", path)
	return db
}

// defaultDBPath keeps the database out of the working tree, so running the
// server from the repo never leaves an untracked file behind.
func defaultDBPath() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ".ckad-simulator", "sessions.db"), nil
}
