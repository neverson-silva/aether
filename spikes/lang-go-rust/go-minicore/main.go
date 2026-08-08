package main

import (
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	ch := make(chan struct{})
	for i := 0; i < 12; i++ {
		go func() { <-ch }()
	}

	cache := make(map[uint64]string, 10000)
	for i := uint64(0); i < 10000; i++ {
		cache[i] = strings.Repeat("x", 16)
	}
	_ = cache

	l, err := net.Listen("tcp", "127.0.0.1:18080")
	if err != nil {
		fmt.Println("listen err", err)
		os.Exit(1)
	}
	fmt.Println("READY", os.Getpid())
	_ = l
	for {
		time.Sleep(time.Hour)
	}
}
