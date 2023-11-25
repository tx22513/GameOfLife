package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
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

func (b *Broker) AggregateCellNumbers(req *stubs.Request, aggregatedRes *stubs.Response) error {
	var wg sync.WaitGroup
	totalCells := 0
	latestTurn := 0

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

			partRes := new(stubs.Response)
			err = client.Call(stubs.SendCellNumber, &stubs.Request{}, partRes)
			if err != nil {
				log.Printf("Error calling SendCellNumber on server %s: %v", addr, err)
			} else {
				b.dataLock.Lock()
				totalCells += partRes.Cellnum
				if partRes.Turn > latestTurn {
					latestTurn = partRes.Turn // 更新最新的回合数
				}
				b.dataLock.Unlock()
			}
		}(addr)
	}

	wg.Wait()

	// 设置聚合的结果
	aggregatedRes.Cellnum = totalCells
	aggregatedRes.Turn = latestTurn // 使用最新的回合数

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
