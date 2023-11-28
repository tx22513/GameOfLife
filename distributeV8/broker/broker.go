package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"sync"
	"uk.ac.bris.cs/gameoflife/goUtils"
	"uk.ac.bris.cs/gameoflife/stubs"
)

func countCell(world [][]uint8) int {
	counter := 0
	for _, row := range world {
		for _, value := range row {
			if value == 255 {
				counter++
			}
		}
	}
	return counter
}

type Broker struct {
	params      goUtils.Params
	world       [][]uint8
	turn        int
	dataLock    sync.Mutex
	serverAddrs []string
}

func (b *Broker) ShutDownAllServers(req *stubs.Request, res *stubs.Response) error {
	var wg sync.WaitGroup

	for _, addr := range b.serverAddrs {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			client, err := rpc.Dial("tcp", addr)
			if err != nil {
				log.Printf("Error connecting to server %s: %v", addr, err)
				return
			}
			defer client.Close()

			shutDownRes := new(stubs.Response)
			err = client.Call(stubs.ShotDown, req, shutDownRes)
			if err != nil {
				log.Printf("Error calling ShotDown on server %s: %v", addr, err)
			}
		}(addr)
	}

	wg.Wait()
	os.Exit(0)
	return nil

}

func (b *Broker) PauseAllServers(req *stubs.Request, res *stubs.Response) error {
	var wg sync.WaitGroup

	for _, addr := range b.serverAddrs {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			client, err := rpc.Dial("tcp", addr)
			if err != nil {
				log.Printf("Error connecting to server %s: %v", addr, err)
				return
			}
			defer client.Close()

			pauseRes := new(stubs.Response)
			err = client.Call(stubs.Pause, req, pauseRes)
			if err != nil {
				log.Printf("Error calling Pause on server %s: %v", addr, err)
			}
		}(addr)
	}

	wg.Wait()

	return nil
}

func (b *Broker) UnPauseAllServers(req *stubs.Request, res *stubs.Response) error {
	var wg sync.WaitGroup

	for _, addr := range b.serverAddrs {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			client, err := rpc.Dial("tcp", addr)
			if err != nil {
				log.Printf("Error connecting to server %s: %v", addr, err)
				return
			}
			defer client.Close()

			unpauseRes := new(stubs.Response)
			err = client.Call(stubs.UnPause, req, unpauseRes)
			if err != nil {
				log.Printf("Error calling UnPause on server %s: %v", addr, err)
			}
		}(addr)
	}

	wg.Wait()

	return nil
}

func (b *Broker) DisconnectAllServers(req *stubs.Request, res *stubs.Response) error {
	var wg sync.WaitGroup

	for _, addr := range b.serverAddrs {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			client, err := rpc.Dial("tcp", addr)
			if err != nil {
				log.Printf("Error connecting to server %s: %v", addr, err)
				return
			}
			defer client.Close()

			disconnectRes := new(stubs.Response)
			err = client.Call(stubs.DisconnectClient, req, disconnectRes)
			if err != nil {
				log.Printf("Error calling DisconnectClient on server %s: %v", addr, err)
			}
		}(addr)
	}

	wg.Wait()

	return nil
}
func (b *Broker) AggregateCurrentState(req *stubs.Request, aggregatedRes *stubs.Response) error {
	var wg sync.WaitGroup
	partialWorlds := make([][][]uint8, len(b.serverAddrs))
	var firstTurn int  // 用于存储第一个成功响应的服务器的回合数
	var turnFound bool // 用于标记是否已找到有效的回合数

	for i, addr := range b.serverAddrs {
		wg.Add(1)
		go func(index int, addr string) {
			defer wg.Done()
			client, err := rpc.Dial("tcp", addr)
			if err != nil {
				log.Printf("Error connecting to server %s: %v", addr, err)
				return
			}
			defer client.Close()

			partRes := new(stubs.Response)
			err = client.Call(stubs.SendCurrentState, req, partRes)
			if err != nil {
				log.Printf("Error calling SendCurrentState on server %s: %v", addr, err)
				return
			}

			b.dataLock.Lock()
			defer b.dataLock.Unlock()

			partialWorlds[index] = partRes.World
			if !turnFound && partRes.Turn > 0 {
				firstTurn = partRes.Turn
				turnFound = true
			}
		}(i, addr)
	}

	wg.Wait()

	// Aggregate partial worlds
	var completeWorld [][]uint8
	for _, part := range partialWorlds {
		if part != nil {
			completeWorld = append(completeWorld, part...)
		}
	}

	aggregatedRes.World = completeWorld
	aggregatedRes.Turn = firstTurn // 设置聚合后的回合数

	return nil
}

func (b *Broker) AggregateCellNumbers(req *stubs.Request, aggregatedRes *stubs.Response) error {
	// 这里我们假设 serverAddr1 对应于处理整个世界状态的服务器
	serverAddr := b.serverAddrs[0] // 使用第一个服务器地址
	client, err := rpc.Dial("tcp", serverAddr)
	if err != nil {
		log.Printf("Error connecting to server %s: %v", serverAddr, err)
		return err
	}
	defer client.Close()

	partRes := new(stubs.Response)
	err = client.Call(stubs.SendCellNumber, &stubs.Request{}, partRes)
	if err != nil {
		log.Printf("Error calling SendCellNumber on server %s: %v", serverAddr, err)
		return err
	}

	// 使用服务器1返回的细胞数和回合数
	aggregatedRes.Cellnum = partRes.Cellnum
	aggregatedRes.Turn = partRes.Turn

	return nil
}

func (b *Broker) CallServerProcessWorld(req *stubs.Request, aggregatedRes *stubs.Response) error {
	var wg sync.WaitGroup

	partialResults := make([]*stubs.Response, len(b.serverAddrs))
	for i, addr := range b.serverAddrs {
		wg.Add(1)
		go func(index int, addr string) {
			defer wg.Done()
			client, err := rpc.Dial("tcp", addr)
			if err != nil {
				log.Printf("Error connecting to server %s: %v", addr, err)
				return
			}
			defer client.Close()

			startRow := index * (len(req.World) / len(b.serverAddrs))
			endRow := startRow + (len(req.World) / len(b.serverAddrs))
			if index == len(b.serverAddrs)-1 {
				endRow = len(req.World) // 确保最后一个 Server 处理所有剩余的行
			} else {
				endRow = endRow // 保证不重叠
			}
			// 创建新的请求对象
			serverReq := &stubs.Request{
				World:    req.World,
				Params:   req.Params,
				StartRow: startRow,
				EndRow:   endRow,
			}

			partRes := new(stubs.Response)
			err = client.Call(stubs.Update, serverReq, partRes)
			if err != nil {
				log.Printf("Error calling server %s: %v", addr, err)
			} else {
				partialResults[index] = partRes
			}
		}(i, addr)
	}

	wg.Wait()

	// 将每个服务器的结果部分拼接到一起
	var aggregatedWorld [][]uint8
	for _, partRes := range partialResults {
		if partRes != nil {

			aggregatedWorld = append(aggregatedWorld, partRes.World...)
		}
	}
	// 将聚合后的世界状态放入 aggregatedRes 中

	aggregatedRes.World = aggregatedWorld
	aggregatedRes.StateChanges = partialResults[0].StateChanges
	fmt.Println(partialResults[0].StateChanges)
	for _, partRes := range partialResults {
		if partRes != nil {
			aggregatedRes.Turn = partRes.Turn

			break // 因为所有服务器的 turn 应该相同
		}
	}
	return nil
}

func (b *Broker) LoadWorldToBroker(req *stubs.Request, res *stubs.Response) (err error) {
	fmt.Println("LoadWorldToBroker")

	if req == nil {
		err = errors.New("Empty data")
		return
	}

	fmt.Println("Load World data...")

	b.dataLock.Lock()
	b.world = req.World

	b.params = req.Params
	b.turn = 0
	b.dataLock.Unlock()

	return

}

func main() {
	pAddr := flag.String("port", "8034", "Port to listen on")
	serverAddr1 := flag.String("serverAddr1", "127.0.0.1:8035", "Server address")
	serverAddr2 := flag.String("serverAddr2", "127.0.0.1:8036", "Server address")
	flag.Parse()
	flag.Parse()
	worker := &Broker{
		serverAddrs: []string{*serverAddr1, *serverAddr2},
	}
	err := rpc.Register(worker)
	if err != nil {
		log.Fatalf("Error registering service: %v", err)
	}
	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Println("listening on %s", listener.Addr().String())
	defer listener.Close()
	rpc.Accept(listener)

}
