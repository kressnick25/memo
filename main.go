package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
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
		log.Println("Usage: memo <program to run> <program args>")
	}

	args := os.Args[1:]
	cmdString := strings.Join(args, " ")
	cmdHash := hash(cmdString)

	cache := db.Cache{Path: os.Getenv("HOME") + "/.memo"}

	err := cache.Setup()
	if err != nil {
		log.Fatalf("Error setting up cache: %s", err.Error())
	}

	cacheEntry, err := cache.Get(cmdHash)
	if err != nil {
		log.Fatalf("error retreiving cache data: %s", err.Error())
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
		log.Fatalf("error executing supplied command: %s", err.Error())
	}

	err = cache.Store(cmdHash, stdout)
	if err != nil {
		log.Fatalf("error storing cache data: %s", err.Error())
	}

	// write output of cmd
	_, err = os.Stdout.Write(stdout)
	check(err)

	os.Exit(0)
}
