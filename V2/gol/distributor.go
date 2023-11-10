package gol

import (
	"fmt"
	"os"
	"time"
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

	// TODO: Create a 2D slice to store the world.
	world := createNewWorld(p)
	world = loadWorld(p, c)

	ticker := time.NewTicker(time.Second * 2) //Event should be sent every 2s.
	defer ticker.Stop()

	turn := 0
	for turn < p.Turns {
		select {
		case <-ticker.C:
			c.events <- AliveCellsCount{turn, countCell(world)}

		case key := <-c.keyPresses:
			handleKeyPress(key, c, world, turn)

		default:
			turn++
			world = executeTurn(p, c, world)
		}

	}

	finalizeGame(p, c, world)

}

// handle different keys
func handleKeyPress(key rune, c distributorChannels, world [][]uint8, turn int) {
	switch key {
	case 's':
		//handle save command
		c.ioCommand <- ioOutput
		fileName := fmt.Sprintf("output_%d", turn)
		c.ioFilename <- fileName
		for _, row := range world {
			for _, value := range row {
				c.ioOutput <- value
			}
		}
		c.events <- ImageOutputComplete{turn, fileName}
		fmt.Println("Saved current state to PGM image.")
	case 'q':
		// handle quit command
		c.events <- StateChange{turn, Quitting}
		c.ioCommand <- ioOutput
		c.ioFilename <- fmt.Sprintf("output_%d", turn)
		for _, row := range world {
			for _, value := range row {
				c.ioOutput <- value
			}
		}
		fmt.Println("Saved current state to PGM image and quit.")
		os.Exit(0)

	case 'p':
		//handle pause command
		c.events <- StateChange{turn, Paused}
		for {
			tem := <-c.keyPresses
			if tem == 'p' {
				c.events <- StateChange{turn, Executing}
				break
			}
		}

	}
}

// count the number of cells
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

// TODO: Execute all turns of the Game of Life.
func executeTurn(p Params, c distributorChannels, world [][]uint8) [][]uint8 {
	res := createNewWorld(p)

	if p.Threads == 1 {
		res = calculateNextState(p, 0, p.ImageHeight, 0, p.ImageWidth, world, c, p.Turns)
		c.events <- TurnComplete{p.Turns + 1}
	} else {
		outChan := make([]chan [][]uint8, p.Threads)
		for i := 0; i < p.Threads; i++ {
			outChan[i] = make(chan [][]uint8)
		}
		for i := 0; i < p.Threads; i++ {
			go worker(p, i*p.ImageHeight/p.Threads, (i+1)*p.ImageHeight/p.Threads, 0, p.ImageWidth, world, outChan[i], c, p.Turns+1)
		}
		res = nil
		for i := 0; i < p.Threads; i++ {
			part := <-outChan[i]
			res = append(res, part...)
		}
		c.events <- TurnComplete{p.Turns + 1}
	}

	return res
}
func worker(p Params, startY, endY, startX, endX int, world [][]uint8, outChan chan<- [][]uint8, c distributorChannels, turn int) {
	outChan <- calculateNextState(p, startY, endY, startX, endX, world, c, turn)
}
func createNewPiece(height, width int) [][]uint8 {
	newWorld := make([][]uint8, height)
	for v := range newWorld {
		newWorld[v] = make([]uint8, width)
	}
	return newWorld
}

func finalizeGame(p Params, c distributorChannels, world [][]uint8) {
	// TODO: Report the final state using FinalTurnCompleteEvent.
	//output
	fileName := fmt.Sprintf("output_%d", p.Turns)
	c.ioCommand <- ioOutput
	c.ioFilename <- fileName
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}

	c.events <- ImageOutputComplete{p.Turns, fileName}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}
	c.events <- FinalTurnComplete{p.Turns, calculateAliveCells(world)}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

// load world data
func loadWorld(p Params, c distributorChannels) [][]uint8 {
	res := createNewWorld(p)
	c.ioCommand <- ioInput
	filename := fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	c.ioFilename <- filename
	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			value := <-c.ioInput
			res[i][j] = value
		}
	}
	return res
}

// create a new world
func createNewWorld(p Params) [][]uint8 {
	newWorld := make([][]uint8, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]uint8, p.ImageWidth)
	}
	return newWorld
}

func calculateNextState(p Params, startY, endY, startX, endX int, world [][]uint8, c distributorChannels, turn int) [][]uint8 {
	newWorld := createNewPiece(endY-startY, p.ImageWidth)

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			aliveNeighbors := countAliveNeighbors(x, y, p, world)
			newWorld[y-startY][x] = nextCellState(x, y, aliveNeighbors, world[y][x], c, turn)
		}
	}
	return newWorld
}

func countAliveNeighbors(x, y int, p Params, world [][]uint8) int {
	alive := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i == 0 && j == 0 {
				continue // Skip the cell itself
			}
			neighborX := (x + i + p.ImageWidth) % p.ImageWidth
			neighborY := (y + j + p.ImageHeight) % p.ImageHeight
			if world[neighborY][neighborX] == 255 {
				alive++
			}
		}
	}
	return alive
}

func nextCellState(x, y, aliveNeighbors int, currentState uint8, c distributorChannels, turn int) uint8 {
	// Rules for cell state in the Game of Life
	switch {
	case currentState == 255 && aliveNeighbors < 2:
		c.events <- CellFlipped{turn, util.Cell{X: y, Y: x}}
		return 0 // Underpopulation
	case currentState == 255 && (aliveNeighbors == 2 || aliveNeighbors == 3):
		return 255 // Lives on
	case currentState == 255 && aliveNeighbors > 3:
		c.events <- CellFlipped{turn, util.Cell{X: y, Y: x}}
		return 0 // Overpopulation
	case currentState == 0 && aliveNeighbors == 3:
		return 255 // Reproduction
	default:
		return currentState
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
