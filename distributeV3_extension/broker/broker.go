package broker

import (
	"log"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/goUtils"
	"uk.ac.bris.cs/gameoflife/stubs"
)

type WorldPart struct {
	StartRow int
	EndRow   int
	Data     [][]uint8
}

type Broker interface {
	DivideWorld(world [][]uint8, parts int) []WorldPart
	AssignPartsToServers(parts []WorldPart, servers []string) error
	CollectResults(parts []WorldPart) [][]uint8
}

type GameOfLifeBroker struct {
	Parts []WorldPart // 用于存储分割的游戏世界部分
}

func (g *GameOfLifeBroker) AssignPartsToServers(world [][]byte, servers []string, p goUtils.Params) error {
	numServers := len(servers)
	imageSize := len(world)
	splitSize := imageSize / numServers
	diff := imageSize % numServers

	pos := 0
	for n := 0; n < numServers; n++ {
		startRow := pos
		pos += splitSize
		if diff != 0 {
			pos++
			diff--
		}
		endRow := pos

		part := WorldPart{
			Data:     world[startRow:endRow],
			StartRow: startRow,
			EndRow:   endRow,
		}

		g.Parts[n] = part

		server := servers[n]
		client, err := rpc.Dial("tcp", server)
		if err != nil {
			return err
		}
		defer client.Close()

		err = client.Call(stubs.LoadWorld, stubs.Request{
			World:    part.Data,
			Params:   p,
			StartRow: part.StartRow,
			EndRow:   part.EndRow,
		}, nil)
		if err != nil {
			return err
		}
	}
	return nil
}

// create a new world
func createNewWorld(p goUtils.Params) [][]uint8 {
	newWorld := make([][]uint8, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]uint8, p.ImageWidth)
	}
	return newWorld
}
func (g *GameOfLifeBroker) CollectResults(servers []string, p goUtils.Params) ([][]uint8, int) {
	finalWorld := createNewWorld(p)
	maxTurn := 0
	var wg sync.WaitGroup
	parts := g.Parts

	responseCh := make(chan stubs.Response, len(parts)) // 创建一个响应通道

	// 并发地从每个服务器收集结果
	for i, part := range parts {
		wg.Add(1)
		go func(part WorldPart, server string) {
			defer wg.Done()

			client, err := rpc.Dial("tcp", server)
			if err != nil {
				log.Printf("Dialing failed: %v", err)
				return
			}
			defer client.Close()

			var response stubs.Response
			err = client.Call(stubs.Update, part, &response)
			if err != nil {
				log.Printf("RPC call failed: %v", err)
				return
			}

			responseCh <- response
		}(part, servers[i%len(servers)])
	}

	go func() {
		wg.Wait()
		close(responseCh) // 当所有goroutine完成时关闭通道
	}()

	// 收集结果
	for response := range responseCh {
		for rowIdx, row := range response.World {
			finalRowIndex := response.StartRow + rowIdx
			if finalRowIndex < len(finalWorld) {
				finalWorld[finalRowIndex] = row
			} else {
				log.Printf("Warning: Trying to access index %d which is out of range in finalWorld", finalRowIndex)
			}
		}

		if response.Turn > maxTurn {
			maxTurn = response.Turn
		}
	}

	return finalWorld, maxTurn
}
