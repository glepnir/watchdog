// Copyright 2020 glepnir. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"reflect"
	"syscall"
	"time"
	"unsafe"
)

var (
	command  string
	filename string
)

func main() {
	flag.StringVar(&command, "c", "run", "exec input command.")
	flag.StringVar(&filename, "f", "main", "the filename of watch.")
	flag.Parse()
	runflag := true
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchStart(ctx, runflag)
	<-quit
}

func currentPath() string {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal("error")
	}
	return dir
}

func watchStart(ctx context.Context, runflag bool) {
	fullname := currentPath() + "/" + filename + ".go"
	done := make(chan struct{})
	go watchFile(fullname, done, ctx)
	go run(ctx, command, fullname, done, runflag)
	select {
	case <-ctx.Done():
		break
	}
}

func run(ctx context.Context, command, fullname string, done chan struct{}, runflag bool) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			if runflag {
				execCommand(command, fullname)
				runflag = false
			} else {
				<-done
				execCommand(command, fullname)
			}
		}
	}
}

func execCommand(command, fullname string) {
	cmd := exec.Command("go", command, fullname)
	stdout, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
		return
	}
	fmt.Println(ByteSliceToString(stdout))
}

func watchFile(file string, done chan struct{}, ctx context.Context) {
	initialstate, err := os.Stat(file)
	if err != nil {
		log.Fatal(err)
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			currentstate, err := os.Stat(file)
			if err != nil {
				log.Fatal(err)
				return
			}
			if currentstate.Size() != initialstate.Size() || currentstate.ModTime() != initialstate.ModTime() {
				done <- struct{}{}
			}
			time.Sleep(1 * time.Second)
			initialstate = currentstate
		}
	}
}

func ByteSliceToString(b []byte) string {
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	sh := reflect.StringHeader{Data: bh.Data, Len: bh.Len}
	return *(*string)(unsafe.Pointer(&sh))
}
