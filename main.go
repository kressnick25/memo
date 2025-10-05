package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"internal/db"
)

const helpText string = 
`Usage: memo <program to run> <program args>

memo is a program to cache the stdout of other command-line programs.
This can be useful if you have cli program that takes a while to execute,
but the result does not change very often.

For example:
memo 'echo "scale=2000; 4*a(1)" | bc -l'

This will take a while to compute pi to 2000 places the first time it runs.
But after memo has cached the result, subsequent runs will return instantaneously.

Memo has a default TTL of 3600 seconds or one hour. This can be customised with the '-t' or '--ttl' flag.
e.g. memo --ttl=24h my-command

Debug logs can be enabled by setting the MEMO_DEBUG environment variable to 'true' or '1'.
`

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

type Args struct {
	Clear bool
	Ttl time.Duration
}

func parseArgs(defaultArgs *Args, args []string) error {
	for _, pair := range args {
		kv := strings.Split(pair, "=")
		k := kv[0]
		key := strings.ReplaceAll(k, "-", "")
		val := kv[1]
		if strings.ToLower(key) == "clear" {
			c, err := strconv.ParseBool(val)
			if err != nil {
				return fmt.Errorf("Unable to parse value of 'clear' argument as boolean: %s", val)	
			}
			slog.Debug("MemoArg Clear set", "clear", c)
			defaultArgs.Clear = c
		} else if strings.ToLower(key) == "ttl" {
			d, err := time.ParseDuration(val)
			if err != nil {
				return fmt.Errorf("Unable to parse value of 'ttl' argument as duration: %s, error: %w", val, err)	
			}
			slog.Debug("MemoArg TTL set", "duration", d.String())
			defaultArgs.Ttl = d
		} else {
			slog.Warn("Unknown argument", "arg", key)
		}
	}

	return nil
}

// find the first arg after '$ memo' to the first instance of a non-memo argument
func findArgs(args []string) []string {
	result := []string{}
	for _, arg := range args {
		if arg[0] == '-' && strings.Contains(arg, "=") {
			result = append(result, arg)
		} else {
			break
		}
	}
	return result
}

func main() {
	// Args
	if len(os.Args) < 2 {
		slog.Error(helpText)
		os.Exit(1)
	}

	if os.Getenv("MEMO_DEBUG") != "" {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	}

	var memoArgs Args
	durationResult, _ := time.ParseDuration("1h")
	memoArgs = Args{
		Clear: false,
		Ttl: durationResult,
	}

	args := os.Args[1:]
	// if first arg is not a subcommand, but a memo arg
	if args[0][0] == '-' {
		m := findArgs(args)	
		var err error
		err = parseArgs(&memoArgs, m)
		if err != nil {
			slog.Error("Error parsing memo args", "err", err)
		}
		// remove the memor args from the slice
		args = args[len(m):]
	}
	cmdString := strings.Join(args, " ")
	cmdHash := hash(cmdString)

	cache := db.Cache{Path: os.Getenv("HOME") + "/.memo"}

	// Setup cache
	err := cache.Setup()
	if err != nil {
		slog.Error("Error setting up cache", "err", err)
		os.Exit(1)
	}

	if memoArgs.Clear {
		err = cache.Clear()
		if err != nil {
			slog.Error("error clearing cache", "err", err)
		}
		slog.Info("memo cache cleared")
	}

	if len(args) > 0 {
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
			err = cache.Store(cmdHash, memoArgs.Ttl, stdout)
			if err != nil {
				slog.Error("error storing cache data", "err", err)
			}
		}()

		// write output of cmd
		_, err = os.Stdout.Write(stdout)
		check(err)

		wg.Wait()
	}
	os.Exit(0)
}
