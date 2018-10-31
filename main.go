package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/sh0rez/luxtronik2-exporter/pkg/luxtronik"
)

func main() {
	lux := luxtronik.Connect("172.21.20.103")

	var wg sync.WaitGroup
	wg.Add(1)
	go lux.Refresh(&wg)

	for {
		fmt.Println("HD:", lux.Value("eingänge", "hd"))
		time.Sleep(time.Second)
	}
}
