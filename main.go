package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
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
		println("Usage: memo <program to run> <program args>")
		os.Exit(1)
	}

	args := os.Args[1:]
	cmdString := strings.Join(args, " ")
	cmdHash := hash(cmdString)

	cache := db.Cache{Path: os.Getenv("HOME") + "/.memo"}

	err := cache.Setup()
	if err != nil {
		println(err.Error())
	}

	cachedOutput, err := cache.Get(cmdHash)
	check(err)
	if cachedOutput != nil {
		_, err = io.Copy(os.Stdout, cachedOutput)
		check(err)
		return
	}

	// exec supplied command
	cmd := buildCommand(args)
	cmd.Env = os.Environ()

	stdout, err := cmd.Output()
	if err != nil {
		fmt.Printf("error executing supplied command: %s\n", err.Error())
	}

	err = cache.Store(cmdHash, stdout)
	check(err)

	// write output of cmd
	_, err = os.Stdout.Write(stdout)
	check(err)
}
