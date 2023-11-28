package main

import (
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
	"uk.ac.bris.cs/gameoflife/util"
)

func extractRows(world [][]uint8, startRow, endRow int) [][]uint8 {
	var extracted [][]uint8
	for i := startRow; i < endRow; i++ {
		if i < len(world) {
			extracted = append(extracted, world[i])
		}
	}
	return extracted
}
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
	newWorld := make([][]uint8, height)
	for v := range newWorld {
		newWorld[v] = make([]uint8, width)
	}
	return newWorld
}

func calculateNextState(p goUtils.Params, startY, endY, startX, endX int, world [][]uint8, turn int) ([][]uint8, []stubs.CellStateChange) {
	newWorld := CreateNewWorld(endY-startY, p.ImageWidth)
	var stateChanges []stubs.CellStateChange

	if endY > len(world) {
		endY = len(world)
	}

	for y := startY; y < endY; y++ {
		for x := startX; x < endX; x++ {
			aliveNeighbors := countAliveNeighbors(x, y, p, world)
			newState := nextCellState(aliveNeighbors, world[y][x])
			newWorld[y-startY][x] = newState
			if newState != world[y][x] {
				stateChange := stubs.CellStateChange{Cell: util.Cell{X: x, Y: y}, Turn: turn}
				stateChanges = append(stateChanges, stateChange)
			}
		}
	}

	return newWorld, stateChanges
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
	startRow     int
	endRow       int
	stateChanges []stubs.CellStateChange
	isPause      bool
	isShotdown   bool
	dataLock     sync.Mutex
}

func (s *Server) ShotDown(req *stubs.Request, res *stubs.Response) (err error) {
	done := make(chan bool, 1)
	go func() {
		fmt.Println("Server is shutting down...")
		time.Sleep(time.Second) // 给足够的时间来处理任何待处理的请求
		done <- true
	}()
	<-done
	s.isShotdown = true
	os.Exit(0)
	return nil
}

func (s *Server) DisconnectClient(req *stubs.Request, res *stubs.Response) (err error) {

	fmt.Println("Client disconnected")

	return
}

func (s *Server) UnPause(req *stubs.Request, res *stubs.Response) (err error) {

	s.isPause = false
	fmt.Println("UnPause")
	return
}

func (s *Server) Pause(req *stubs.Request, res *stubs.Response) (err error) {

	s.isPause = true

	fmt.Println("Pause")
	return
}

func (s *Server) SendCurrentState(req *stubs.Request, res *stubs.Response) (err error) {

	s.dataLock.Lock()
	res.World = s.world
	res.Turn = s.turn
	s.dataLock.Unlock()
	fmt.Println("Sending Current State...")
	return
}
func (s *Server) SendCellNumber(req *stubs.Request, res *stubs.Response) (err error) {

	num := countCell(s.world)

	fmt.Println("Sending cell number for specified rows...")
	res.Cellnum = num
	res.Turn = s.turn

	return
}

func (s *Server) Update(req *stubs.Request, res *stubs.Response) (err error) {

	fmt.Println("Loading...")
	s.dataLock.Lock()
	s.world = req.World
	s.startRow = req.StartRow
	s.endRow = req.EndRow
	s.params = req.Params
	s.stateChanges = []stubs.CellStateChange{}
	s.turn = 0

	s.dataLock.Unlock()

	fmt.Println("Updating ...")
	params := s.params
	world := s.world

	for s.turn < params.Turns && !s.isShotdown {

		s.dataLock.Lock()
		if s.isPause {
			s.dataLock.Unlock()
			//wait isPause become false
			for {
				s.dataLock.Lock()
				if !s.isPause {
					s.dataLock.Unlock()
					break
				}
				s.dataLock.Unlock()

				time.Sleep(100 * time.Millisecond)
			}
		} else {
			s.dataLock.Unlock()
		}

		var changes []stubs.CellStateChange
		world, changes = calculateNextState(params, 0, params.ImageHeight, 0, params.ImageWidth, world, s.turn)
		s.stateChanges = append(s.stateChanges, changes...)
		s.turn++
		s.world = world
		fmt.Println(s.stateChanges)

	}

	res.Turn = s.turn

	res.World = extractRows(s.world, s.startRow, s.endRow)
	res.StateChanges = s.stateChanges

	res.StartRow = s.startRow
	res.EndRow = s.endRow

	return
}

func main() {
	pAddr := flag.String("port", "8036", "Port to listen on")
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
