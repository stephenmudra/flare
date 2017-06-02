package main

import (
	"os/signal"
	"os"
	"log"

	"github.com/joyrexus/buckets"
)

func main() {
	bx, err := buckets.Open("Flare.db")
	if (err != nil) {
		panic(err)
	}
	defer bx.Close()

	db, err := bx.New([]byte("FlareV1"))
	if (err != nil) {
		panic(err)
	}

	rest := RestApi{"8080", db}
	go rest.serve()

	dns := DnsServer{"53", db}
	go dns.tcpServe()
	go dns.udpServe()

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt, os.Kill)
	for {
		select {
		case s := <-sig:
			log.Fatalf("fatal: signal %s received\n", s)
		}
	}
}
