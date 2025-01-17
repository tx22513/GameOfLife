package gol

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/goUtils"
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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	//connect to the server
	//client, err := rpc.Dial("tcp", "34.229.9.86:8030")
	client, err := rpc.Dial("tcp", "127.0.0.1:8034")
	if err != nil {
		log.Fatal(err)
	}

	// TODO: Create a 2D slice to store the world.
	world := createNewWorld(p.ImageHeight, p.ImageWidth)
	world = loadWorld(p, c)
	turn := 0
	request := stubs.Request{World: world, Params: convertParams(p)}
	response := new(stubs.Response)
	err = client.Call(stubs.LoadWorldToBroker, request, response)

	ticker := time.NewTicker(time.Second * 2) //Event should be sent every 2s.

	defer ticker.Stop()
	go func() {
		for {
			select {
			case <-ticker.C:
				req := stubs.Request{}
				res := new(stubs.Response)
				err = client.Call(stubs.AggregateCellNumbers, req, res)
				c.events <- AliveCellsCount{res.Turn, res.Cellnum}

				err = client.Call(stubs.AggregateCellFlip, req, res)
				for _, change := range res.StateChanges {
					c.events <- CellFlipped{change.Turn, change.Cell}
				}

				c.events <- TurnComplete{res.Turn}

			case key := <-c.keyPresses:
				req := stubs.Request{}
				res := new(stubs.Response)
				err = client.Call(stubs.AggregateCurrentState, req, res)
				handleKeyPress(p, key, c, res.World, res.Turn, client, req, res)

			}

		}

	}()

	//excute turns
	req := stubs.Request{World: world, Params: convertParams(p)}
	res := new(stubs.Response)
	err = client.Call(stubs.CallServerProcessWorld, req, res)

	turn = res.Turn
	world = res.World

	finalizeGame(p, c, world, turn)

}

func handleKeyPress(p Params, key rune, c distributorChannels, world [][]uint8, turn int, client *rpc.Client, req stubs.Request, res *stubs.Response) {
	switch key {
	case 's':
		//handle save command
		c.ioCommand <- ioOutput
		fileName := fmt.Sprintf("output_%d", turn)
		c.ioFilename <- fileName
		for y := 0; y < p.ImageHeight; y++ {
			for x := 0; x < p.ImageWidth; x++ {
				c.ioOutput <- world[y][x]
			}
		}
		c.events <- ImageOutputComplete{turn, fileName}
		fmt.Println("Saved current state to PGM image.")
	case 'q':
		client.Call(stubs.DisconnectAllServers, req, res)
		c.ioCommand <- ioCheckIdle
		<-c.ioIdle
		c.events <- StateChange{res.Turn, Quitting}
		close(c.events)
		os.Exit(0)
	case 'p':
		c.events <- StateChange{res.Turn, Paused}
		client.Call(stubs.PauseAllServers, req, res)
		for {
			tem := <-c.keyPresses
			if tem == 'p' {
				client.Call(stubs.UnPauseAllServers, req, res)
				c.events <- StateChange{res.Turn, Executing}
				break
			}
		}

	case 'k':
		fileName := fmt.Sprintf("%dx%dx%d", p.ImageHeight, p.ImageWidth, res.Turn)
		c.ioCommand <- ioOutput
		c.ioFilename <- fileName
		for y := 0; y < p.ImageHeight; y++ {
			for x := 0; x < p.ImageWidth; x++ {
				c.ioOutput <- world[y][x]
			}
		}

		c.events <- ImageOutputComplete{res.Turn, fileName}
		client.Call(stubs.ShutDownAllServers, req, res)
		c.ioCommand <- ioCheckIdle
		<-c.ioIdle
		c.events <- StateChange{res.Turn, Quitting}
		close(c.events)
		os.Exit(0)
	}
}

func calculateAliveCells(world [][]uint8) []util.Cell {
	cells := []util.Cell{}
	for i := range world {
		for j := range world[i] {
			if world[i][j] == 255 {
				cells = append(cells, util.Cell{X: j, Y: i})
			}
		}
	}
	return cells
}

func finalizeGame(p Params, c distributorChannels, world [][]uint8, turn int) {
	// TODO: Report the final state using FinalTurnCompleteEvent.
	//output
	fileName := fmt.Sprintf("%dx%dx%d", p.ImageHeight, p.ImageWidth, turn)
	c.ioCommand <- ioOutput
	c.ioFilename <- fileName
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}

	c.events <- ImageOutputComplete{turn, fileName}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}
	c.events <- FinalTurnComplete{turn, calculateAliveCells(world)}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

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

func convertParams(p Params) goUtils.Params {
	return goUtils.Params{p.Turns, p.Threads, p.ImageWidth, p.ImageHeight}
}

// create a new world
func createNewWorld(ImageHeight, ImageWidth int) [][]uint8 {
	newWorld := make([][]uint8, ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]uint8, ImageWidth)
	}
	return newWorld
}
