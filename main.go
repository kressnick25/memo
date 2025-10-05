package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"

	"internal/db"
)

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func hash(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func buildCommand(argv []string) exec.Cmd {
	currentShell, isSet := os.LookupEnv("SHELL")
	if !isSet {
		if len(argv) > 0 {
			return *exec.Command(argv[0], strings.Join(argv[0:], " "))
		}
		return *exec.Command(argv[0])
	}

	return *exec.Command(currentShell, "-c", strings.Join(argv[0:], " "))
}

func main() {
	// Args
	if len(os.Args) < 2 {
		slog.Error("Usage: memo <program to run> <program args>")
	}

	if os.Getenv("MEMO_DEBUG") != "" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	args := os.Args[1:]
	cmdString := strings.Join(args, " ")
	cmdHash := hash(cmdString)

	cache := db.Cache{Path: os.Getenv("HOME") + "/.memo"}

	// Setup cache
	err := cache.Setup()
	if err != nil {
		slog.Error("Error setting up cache", "err", err)
		os.Exit(1)
	}

	// Retreive cache entry if exists
	cacheEntry, err := cache.Get(cmdHash)
	if err != nil {
		slog.Error("error retreiving cache data", "err", err)
		os.Exit(1)
	}

	if cacheEntry != nil {
		_, err = io.Copy(os.Stdout, bytes.NewReader(cacheEntry.Data))
		check(err)
		return
	}

	// exec supplied command
	cmd := buildCommand(args)
	cmd.Env = os.Environ()

	stdout, err := cmd.Output()
	if err != nil {
		slog.Error("error executing supplied command", "err", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Store output
		err = cache.Store(cmdHash, stdout)
		if err != nil {
			slog.Error("error storing cache data", "err", err)
		}
	}()

	// write output of cmd
	_, err = os.Stdout.Write(stdout)
	check(err)

	wg.Wait()
	os.Exit(0)
}
