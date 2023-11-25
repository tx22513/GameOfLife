package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
)

var (
	methods  = make(map[string]chan stubs.Data)
	methodmx sync.RWMutex
)

func createMethod(method string, buflen int) {
	methodmx.Lock()
	defer methodmx.Unlock()
	if _, ok := methods[method]; !ok {
		methods[method] = make(chan stubs.Data, buflen)
		fmt.Println("Created channel #", method)
	}
}

func public(method string, data stubs.Data) (err error) {
	methodmx.RLock()
	defer methodmx.RUnlock()
	if ch, ok := methods[method]; ok {
		ch <- data
	} else {
		return errors.New("No such topic.")
	}
	return

}

func subscriber_loop(topic chan stubs.Data, client *rpc.Client, callback string) {
	for {
		job := <-topic
		response := new(stubs.JobReport)
		err := client.Call(callback, job, response)
		if err != nil {
			fmt.Println("Error")
			fmt.Println(err)
			fmt.Println("Closing subscriber thread.")
			//Place the unfulfilled job back on the topic channel.
			topic <- job
			break
		}

	}
}
func subscribe(topic string, serverAddress string, callback string) (err error) {
	fmt.Println("Subscription request")
	methodmx.RLock()
	ch := methods[topic]
	methodmx.RUnlock()
	client, err := rpc.Dial("tcp", serverAddress)
	if err == nil {
		go subscriber_loop(ch, client, callback)
	} else {
		fmt.Println("Error subscribing ", serverAddress)
		fmt.Println(err)
		return err
	}
	return
}

type Broker struct{}

func (b *Broker) CreateChannel(req stubs.ChannelRequest, res *stubs.StatusReport) (err error) {
	createMethod(req.Method, req.Buffer)
	return
}

func (b *Broker) Publish(req stubs.PublishRequest, res *stubs.StatusReport) (err error) {
	// 发布游戏状态
	err = public(req.Method, req.WorldData)
	fmt.Println(req.WorldData)

	return err
}

func (b *Broker) Subscribe(req stubs.Subscription, res *stubs.StatusReport) (err error) {
	err = subscribe(req.Method, req.ServerAddress, req.Callback)
	if err != nil {
		res.Message = "Error during subscription"
	}
	return err
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rpc.Register(&Broker{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}
