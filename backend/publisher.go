package main

import (
	"io/ioutil"
	"log"
	"os"

	stan "github.com/nats-io/stan.go"
)

func main() {
	file, err := os.Open("model.json")
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	sc, _ := stan.Connect("test-cluster", "client-test2", stan.NatsURL("nats://localhost:4222"))
	sc.Publish("foo", data)
	sc.Close()
}
