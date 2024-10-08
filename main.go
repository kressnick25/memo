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

func main() {
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
	var cmd *exec.Cmd
	if len(args) > 1 {
		cmd = exec.Command(args[0], strings.Join(args[1:], " "))
	} else {
		cmd = exec.Command(args[0])
	}

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
