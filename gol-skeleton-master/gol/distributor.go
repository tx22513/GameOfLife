package gol

import (
	"fmt"
	"os"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	keyPresses <-chan rune
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	// TODO: Create a 2D slice to store the world.
	World := createNewWorld(p)
	World = load(p, c)
	turn := 0

	// TODO: Execute all turns of the Game of Life.
	ticker := time.NewTicker(time.Second * 2) //Event should be sent every 2s.
	defer ticker.Stop()
	var mu sync.Mutex

	for true {
		select {
		case <-ticker.C:
			mu.Lock()
			c.events <- AliveCellsCount{turn, countCell(World)}
			mu.Unlock()
		case key := <-c.keyPresses:
			handleKeyPress(key, c, World, turn)

		default:

			var newWorld [][]uint8
			if p.Threads == 1 {
				mu.Lock()
				newWorld = calculateNextState(World)
				mu.Unlock()
			} else {
				outChan := make([]chan [][]uint8, p.Threads)
				workderHight := p.ImageHeight / p.Threads
				for i := 0; i < p.Threads; i++ {
					outChan[i] = make(chan [][]uint8)
					startY := i * workderHight
					endY := startY + workderHight
					go worker(endY, startY, 0, p.ImageWidth, outChan[i])
					fragment := <-outChan[i]
					newWorld = append(newWorld, fragment...)
				}
			}
			World = newWorld
			c.events <- TurnComplete{turn}
			turn++
		}
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.

	// Make sure that the Io has finished any output before exiting.
	fileName := fmt.Sprintf("output_%d", turn)
	c.ioFilename <- fileName
	for _, row := range World {
		for _, value := range row {
			c.ioOutput <- value
		}
	}
	c.events <- ImageOutputComplete{turn, fileName}
	c.events <- FinalTurnComplete{turn, calculateAliveCells(p, World)}
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

// create a new world
func createNewWorld(p Params) [][]uint8 {
	newWorld := make([][]uint8, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]uint8, p.ImageWidth)
	}
	return newWorld
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

func countLiveNeighbour(x int, y int, world [][]uint8) int {
	lst := [8][2]int{

		{-1, 0},
		{-1, -1},
		{-1, 1},
		{0, -1},
		{0, 1},
		{1, -1},
		{1, 0},
		{1, 1},
	}
	count := 0
	for _, r := range lst {
		if world[(x+r[0]+len(world))%len(world)][(y+r[1]+len(world[0]))%len(world[0])] == 255 {
			count++
		}

	}
	return count
}

// any live cell with fewer than two live neighbours dies
// any live cell with two or three live neighbours is unaffected
// any live cell with more than three live neighbours dies
// any dead cell with exactly three live neighbours becomes alive
func calculateNextState(world [][]uint8) [][]uint8 {
	tmp := make([][]uint8, len(world))
	for i := range world {
		tmp[i] = make([]uint8, len(world[i]))
		copy(tmp[i], world[i])
	}
	for x := range world {
		for y := range world[x] {
			count := countLiveNeighbour(x, y, world)
			if world[x][y] == 255 && (count < 2 || count > 3) {
				tmp[x][y] = 0
			} else if world[x][y] == 0 && count == 3 {
				tmp[x][y] = 255
			}
		}
	}
	return tmp
}

// ！！！
func worker(startY, endY, startX, endX int, out chan<- [][]uint8) {
	world := make([][]uint8, endY-startY)
	for i := range world {
		world[i] = make([]uint8, endX-startX)
	}

	out <- calculateNextState(world)
}

func calculateAliveCells(p Params, world [][]uint8) []util.Cell {
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
func load(p Params, c distributorChannels) [][]uint8 {
	c.ioCommand <- ioInput
	World := createNewWorld(p)
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			input := <-c.ioInput
			if input != 0 {
				World[y][x] = input
				c.events <- CellFlipped{0, util.Cell{X: y, Y: x}}
			}
		}
	}
	return World
}
