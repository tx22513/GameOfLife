package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/goUtil"
	"uk.ac.bris.cs/gameoflife/stubs"
)

func nextCellState(aliveNeighbors int, currentState uint8) uint8 {
	// Rules for cell state in the Game of Life
	switch {
	case currentState == 255 && aliveNeighbors < 2:
		return 0 // Underpopulation
	case currentState == 255 && (aliveNeighbors == 2 || aliveNeighbors == 3):
		return 255 // Lives on
	case currentState == 255 && aliveNeighbors > 3:
		return 0 // Overpopulation
	case currentState == 0 && aliveNeighbors == 3:
		return 255 // Reproduction
	default:
		return currentState
	}
}

func countAliveNeighbors(x, y int, p goUtil.Params, world [][]uint8) int {
	alive := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if !(i == 0 && j == 0) {
				neighborX := (x + j + p.ImageWidth) % p.ImageWidth
				neighborY := (y + i + p.ImageHeight) % p.ImageHeight
				if world[neighborY][neighborX] != 0 {
					alive++
				}
			}
		}
	}
	return alive
}

func calculateNextState(p goUtil.Params, startY, endY, startX, endX int, world [][]uint8) [][]uint8 {
	newWorld := goUtil.CreateNewWorld(endY-startY, p.ImageWidth)
	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			aliveNeighbors := countAliveNeighbors(x, y, p, world)
			newState := nextCellState(aliveNeighbors, world[y][x])
			newWorld[y-startY][x] = newState
			if newState != world[y][x] {
			}
		}
	}
	return newWorld
}

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

type Worker struct {
	Params  goUtil.Params
	World   goUtil.World
	turns   int
	isPause bool
	isStop  bool
	test    int
}

func (w *Worker) Load(req stubs.Request, res *stubs.Response) error {

	w.test = req.Num

	return nil
}
func (w *Worker) ADD(req stubs.Request, res *stubs.Response) error {

	res.Cellnum = w.test

	return nil
}

//func (w *Worker) LoadWorld(req stubs.Request) {
//
//	w.World = req.World
//	w.Params = req.Params
//	w.turns = 0
//
//	return
//}
//
//func (w *Worker) SendCellNumber(res *stubs.Response) {
//	world := w.World
//	cellNumber := countCell(world)
//
//	res.Cellnum = cellNumber
//}
//
//func (w *Worker) SendWorld(res *stubs.Response) {
//	world := w.World
//	params := w.Params
//	turn := w.turns
//
//	for params.Turns > turn {
//
//		newWorld := calculateNextState(params, 0, params.ImageHeight, 0, params.ImageWidth, world)
//		turn++
//
//		w.turns = turn
//
//		world = newWorld
//
//		if w.isPause {
//			break
//		}
//	}
//
//	res.World = world
//	res.Turn = turn
//}
//
//func (w *Worker) Pause() {
//	w.isPause = true
//
//}
//func (w *Worker) UnPause() {
//	w.isPause = false
//
//}
//
//func (w *Worker) StopServe() {
//	w.isStop = true
//
//}

func main() {
	pAddr := flag.String("port", "1234", "Port to listen on")
	flag.Parse()
	worker := new(Worker)
	err := rpc.Register(worker)
	if err != nil {
		log.Fatalf("Error registering service: %v", err)
	}
	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	fmt.Printf("listening on %s", listener.Addr().String())
	defer listener.Close()
	rpc.Accept(listener)

}
