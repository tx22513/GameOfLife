package broker

import (
	"log"
	"net/rpc"
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

type GameOfLifeBroker struct{}

func (g *GameOfLifeBroker) DivideWorld(world [][]uint8, parts int) []WorldPart {
	partHeight := len(world) / parts
	var worldParts []WorldPart
	var startRow int

	for i := 0; i < parts; i++ {
		if i == 0 {
			startRow = 0

		} else {
			startRow = i*partHeight + 1
		}

		endRow := (i + 1) * partHeight

		if startRow == 0 {
			part := WorldPart{
				StartRow: startRow,
				EndRow:   endRow,
				Data:     world[startRow : endRow+1],
			}
			worldParts = append(worldParts, part)
		} else if endRow == len(world)-1 {
			part := WorldPart{
				StartRow: startRow,
				EndRow:   endRow,
				Data:     world[startRow-1 : endRow],
			}
			worldParts = append(worldParts, part)
		} else {
			part := WorldPart{
				StartRow: startRow,
				EndRow:   endRow,
				Data:     world[startRow-1 : endRow+1],
			}
			worldParts = append(worldParts, part)
		}

	}

	return worldParts
}

func (g *GameOfLifeBroker) AssignPartsToServers(parts []WorldPart, servers []string, p goUtils.Params) error {

	for i, part := range parts {
		server := servers[i%len(servers)]
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
	newWorld := make([][]uint8, p.ImageHeight+1)
	for i := range newWorld {
		newWorld[i] = make([]uint8, p.ImageWidth)
	}
	return newWorld
}
func (g *GameOfLifeBroker) CollectResults(parts []WorldPart, servers []string, p goUtils.Params) ([][]uint8, int) {
	finalWorld := createNewWorld(p)
	maxTurn := 0

	for i, part := range parts {
		server := servers[i%len(servers)]
		client, err := rpc.Dial("tcp", server)
		if err != nil {
			log.Fatal("Dialing:", err)
		}

		var response stubs.Response
		err = client.Call(stubs.Update, part, &response)
		if err != nil {
			client.Close()
			log.Fatal("RPC call failed:", err)
		}
		client.Close()

		for rowIdx, row := range response.World {
			finalRowIndex := part.StartRow + rowIdx
			if finalRowIndex < len(finalWorld) {
				finalWorld[finalRowIndex] = row
			} else {
				// 这里可以记录错误或者采取其他措施
				log.Printf("Warning: Trying to access index %d which is out of range in finalWorld", finalRowIndex)
			}
		}

		if response.Turn > maxTurn {
			maxTurn = response.Turn
		}
	}

	return finalWorld, maxTurn
}
