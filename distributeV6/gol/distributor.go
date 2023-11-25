package gol

import (
	"flag"
	"fmt"
	"net/rpc"
	"sync"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
	keyPresses <-chan rune
}

// create a new world
func createNewWorld(ImageHeight, ImageWidth int) [][]uint8 {
	newWorld := make([][]uint8, ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]uint8, ImageWidth)
	}
	return newWorld
}

// load world data
func loadWorld(p Params, c distributorChannels) [][]uint8 {
	c.ioCommand <- ioInput
	res := createNewWorld(p.ImageHeight, p.ImageWidth)
	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioFilename <- filename
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			input := <-c.ioInput
			if input != 0 {
				res[y][x] = input
				c.events <- CellFlipped{0, util.Cell{X: y, Y: x}}
			}
		}
	}
	return res
}
func convertParams(p Params) stubs.Params {
	return stubs.Params{p.Turns, p.Threads, p.ImageWidth, p.ImageHeight}
}
func DivideWorld(world [][]uint8, parts int) []stubs.WorldPart {
	partHeight := len(world) / parts
	var worldParts []stubs.WorldPart
	var startRow int

	for i := 0; i < parts; i++ {
		if i == 0 {
			startRow = 0

		} else {
			startRow = i*partHeight + 1
		}

		endRow := (i + 1) * partHeight

		if startRow == 0 {
			part := stubs.WorldPart{
				StartRow: startRow,
				EndRow:   endRow,
				Data:     world[startRow : endRow+1],
			}
			worldParts = append(worldParts, part)
		} else if endRow == len(world) {
			part := stubs.WorldPart{
				StartRow: startRow,
				EndRow:   endRow,
				Data:     world[startRow-1 : endRow],
			}
			worldParts = append(worldParts, part)
		} else {
			part := stubs.WorldPart{
				StartRow: startRow,
				EndRow:   endRow,
				Data:     world[startRow-1 : endRow+1],
			}
			worldParts = append(worldParts, part)
		}

	}

	return worldParts
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	brokerAddr := flag.String("broker", "127.0.0.1:8030", "Address of broker instance")
	client, _ := rpc.Dial("tcp", *brokerAddr)
	status := new(stubs.StatusReport)
	//client.Call(stubs.CreateChannel, stubs.ChannelRequest{Method: "update", Buffer: 10}, status)

	world := createNewWorld(p.ImageHeight, p.ImageWidth)
	world = loadWorld(p, c)
	worldParts := DivideWorld(world, 2)

	var wg sync.WaitGroup
	for _, part := range worldParts {

		wg.Add(1)
		go func(part stubs.WorldPart) {
			defer wg.Done()

			worldData := stubs.Data{P: convertParams(p), Turn: p.Turns, World: part}
			towork := stubs.PublishRequest{Method: "update", WorldData: worldData}
			err := client.Call(stubs.Publish, towork, status)
			if err != nil {
				fmt.Println("RPC client returned error:", err)
			}
		}(part)
	}
	wg.Wait()
	turn := 0

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
