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

	err := cache.Setup()
	if err != nil {
		slog.Error("Error setting up cache", "err", err)
		os.Exit(1)
	}

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

	err = cache.Store(cmdHash, stdout)
	if err != nil {
		slog.Error("error storing cache data", "err", err)
		os.Exit(1)
	}

	// write output of cmd
	_, err = os.Stdout.Write(stdout)
	check(err)

	os.Exit(0)
}
