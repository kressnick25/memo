package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
    "os/exec"
	"strings"
)

const dataDir = "/home/nkress/.memo"

func check(err error) {
    if err != nil {
        panic(err)
    }
}

func fileExists(path string) (bool, error) {
    _, err := os.Stat(path)
    if err == nil { return true, nil }
    if os.IsNotExist(err) { return false, nil }
    return false, err
}

func hash(text string) string {
    hash := md5.Sum([]byte(text))
    return hex.EncodeToString(hash[:])
}

func stream(input *os.File, output *os.File, bufLength int) error {
    buf := make([]byte, bufLength)
	for {
        n, err := input.Read(buf)
        if err != nil {
            if err == io.EOF {
                break
            }
            fmt.Println(err)
            os.Exit(1)
        }
        _, err = output.Write(buf[:n])
        if err != nil {
            return err
        }
	}
    return nil
}

func main() {
    args := os.Args[1:]
    cmdString := strings.Join(args, " ")
    cmdHash := hash(cmdString)


    // ensure Memo directory exists
    exists, err := fileExists(dataDir)
    check(err)
    if !exists {
        err = os.Mkdir(dataDir, os.FileMode(int(0700)))
        if err != nil {
            fmt.Printf("Error creating memo directory '%s': %s\n", dataDir, err.Error())
            os.Exit(1)
        }
    }

    filePath := fmt.Sprintf("%s/%s", dataDir, cmdHash)

    // read cache if exists
    exists, err = fileExists(filePath)
    check(err)
    if exists {
        existingFile, err := os.Open(filePath)
        check(err)
        err = stream(existingFile, os.Stdout, 256)
        check(err) 
        return
    }

    // exec supplied command
    cmd := exec.Command(args[0], strings.Join(args[1:], " "))
    stdout, err := cmd.Output()
    check(err)

    // cache output
    file, err := os.Create(filePath)
    check(err)
    defer file.Close()

    _, err = file.Write(stdout)
    check(err)

    // write output of cmd
    _, err = os.Stdout.Write(stdout)
    check(err)
}
