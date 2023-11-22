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
	"time"
	"uk.ac.bris.cs/gameoflife/goUtils"
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

func countAliveNeighbors(x, y int, p goUtils.Params, world [][]uint8) int {
	alive := 0
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if !(i == 0 && j == 0) {
				neighborX := (x + j + p.ImageWidth) % p.ImageWidth
				neighborY := (y + i + len(world)) % len(world)
				if world[neighborY][neighborX] != 0 {
					alive++
				}
			}
		}
	}
	return alive
}

func CreateNewWorld(height, width int) [][]byte {
	if height < 0 {
		height = 0
	}
	if width < 0 {
		width = 0
	}
	newWorld := make([][]uint8, height)
	for v := range newWorld {
		newWorld[v] = make([]uint8, width)
	}
	return newWorld
}

func calculateNextState(p goUtils.Params, startY, endY, startX, endX int, world [][]uint8) [][]uint8 {

	newWorld := CreateNewWorld(endY-startY, p.ImageWidth)
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

type Server struct {
	params       goUtils.Params
	world        [][]uint8
	turn         int
	initialWorld [][]uint8

	isPause bool

	dataLock sync.Mutex

	StartRow int
	EndRow   int
}

func (s *Server) ShotDown(req *stubs.Request, res *stubs.Response) (err error) {

	os.Exit(0)

	return
}

func (s *Server) DisconnectClient(req *stubs.Request, res *stubs.Response) (err error) {

	fmt.Println("Client disconnected")

	return
}

func (s *Server) UnPause(req *stubs.Request, res *stubs.Response) (err error) {

	s.isPause = false

	return
}

func (s *Server) Pause(req *stubs.Request, res *stubs.Response) (err error) {

	s.isPause = true

	return
}

func (s *Server) SendCurrentState(req *stubs.Request, res *stubs.Response) (err error) {

	s.dataLock.Lock()
	res.World = s.world
	res.Turn = s.turn
	s.dataLock.Unlock()
	return
}

func (s *Server) SendCellNumber(req *stubs.Request, res *stubs.Response) (err error) {
	world := s.world
	turn := s.turn

	num := countCell(world)

	res.Cellnum = num
	res.Turn = turn

	return
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *Server) Update(req *stubs.Request, res *stubs.Response) (err error) {
	params := s.params
	world := s.world
	startRow := max(0, s.StartRow)
	endRow := min(len(s.world), s.EndRow)
	for s.turn < params.Turns {

		s.dataLock.Lock()
		if s.isPause {
			s.dataLock.Unlock()
			// 等待直到 isPause 变为 false
			for {
				s.dataLock.Lock()
				if !s.isPause {
					s.dataLock.Unlock()
					break
				}
				s.dataLock.Unlock()
				// 短暂休眠，避免紧密循环
				time.Sleep(100 * time.Millisecond)
			}
		} else {
			s.dataLock.Unlock()
		}
		world = calculateNextState(params, startRow, endRow, 0, params.ImageWidth, world)
		s.turn++
		s.world = world

	}

	res.Turn = s.turn
	res.World = world
	s.world = world
	res.StartRow = s.StartRow

	return
}

func (s *Server) LoadWorld(req *stubs.Request, res *stubs.Response) (err error) {

	if req == nil {
		err = errors.New("Empty data")
		return
	}

	s.dataLock.Lock()
	s.world = req.World

	s.params = req.Params
	s.turn = 0
	s.StartRow = req.StartRow
	s.EndRow = req.EndRow
	s.dataLock.Unlock()
	return
}

func main() {
	pAddr := flag.String("port", "1235", "Port to listen on")
	flag.Parse()
	worker := new(Server)
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
