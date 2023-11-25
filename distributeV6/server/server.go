package main

import (
	"flag"
	"fmt"
	"net"
	"net/rpc"
	"sync"
	"time"
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

func countAliveNeighbors(x, y int, p stubs.Params, world [][]uint8) int {
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

func calculateNextState(p stubs.Params, startY, endY, startX, endX int, world [][]uint8) [][]uint8 {
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
func getOutboundIP() string {
	conn, _ := net.Dial("udp", "8.8.8.8:80")
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr).IP.String()
	return localAddr
}

type Server struct {
	params       stubs.Params
	world        [][]uint8
	turn         int
	initialWorld [][]uint8

	isPause bool

	dataLock sync.Mutex
}

func (s *Server) Update(req *stubs.Data, res *stubs.JobReport) (err error) {
	fmt.Println("Updating ...")
	s.dataLock.Lock()
	s.world = req.World.Data

	s.params = req.P
	s.turn = 0
	s.dataLock.Unlock()
	params := s.params
	world := s.world
	for s.turn < params.
		Turns {

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
		world = calculateNextState(params, 0, params.ImageHeight, 0, params.ImageWidth, world)
		s.turn++
		s.world = world

	}

	fmt.Println(world)
	res.Turn = s.turn
	res.World = world
	s.world = world

	return
}

func main() {
	pAddr := flag.String("port", "8050", "Port to listen on")
	brokerAddr := flag.String("broker", "127.0.0.1:8030", "Address of broker instance")
	flag.Parse()
	client, _ := rpc.Dial("tcp", *brokerAddr)
	status := new(stubs.StatusReport)
	client.Call(stubs.CreateChannel, stubs.ChannelRequest{Method: "update", Buffer: 10}, status)

	rpc.Register(new(Server))
	fmt.Println(*pAddr)
	listener, err := net.Listen("tcp", ":"+*pAddr)
	if err != nil {
		fmt.Println(err)
	}
	client.Call(stubs.Subscribe, stubs.Subscription{Method: "update", ServerAddress: getOutboundIP() + ":" + *pAddr, Callback: "Server.Update"}, status)

	defer listener.Close()

	rpc.Accept(listener)
	flag.Parse()
}
