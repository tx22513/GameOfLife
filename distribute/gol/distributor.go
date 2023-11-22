package gol

import (
	"fmt"
	"log"
	"net/rpc"
	"uk.ac.bris.cs/gameoflife/goUtil"
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

const serverIP string = "127.0.0.1"
const serverPort string = "1234"

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

//	func makeCall(client *rpc.Client, methodName string, world goUtil.World, params goUtil.Params) (*stubs.Response, error) {
//		request := stubs.Request{World: world, Params: params}
//		response := new(stubs.Response)
//		fmt.Println("Response from call " + methodName)
//		err := client.Call(methodName, request, response)
//
//		if err != nil {
//			// 这里打印错误信息
//			fmt.Printf("调用方法 %s 出错: %s\n", methodName, err)
//			return nil, err
//		}
//
//		// 调试信息，确认收到的响应
//		fmt.Printf("从方法 %s 收到响应: %+v\n", methodName, response)
//
//		return response, err
//	}
func convertParams(p Params) goUtil.Params {
	return goUtil.Params{p.Turns, p.Threads, p.ImageWidth, p.ImageHeight}
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	//world := createNewWorld(p.ImageHeight, p.ImageWidth)
	//world = loadWorld(p, c)
	//connect to server
	//server := fmt.Sprintf("%s:%s", serverIP, serverPort)
	client, err := rpc.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal(err)
	}
	req := stubs.Request{Num: 10}
	response := new(stubs.Response)
	client.Call(stubs.Load, req, &response)
	err = client.Call(stubs.ADD, req, &response)
	fmt.Println(response.Cellnum)
	fmt.Println(err)
	//makeCall(client, stubs.LoadWorld, world, convertParams(p))

	//ticker := time.NewTicker(time.Second * 2)
	//defer ticker.Stop()
	//turn := 0
	//var paused bool
	//for turn < p.Turns {
	//	select {
	//	case <-ticker.C:
	//		res, _ := makeCall(client, stubs.SendCellNumber, world, convertParams(p))
	//		c.events <- AliveCellsCount{turn, res.Cellnum}
	//
	//	case key := <-c.keyPresses:
	//
	//		switch key {
	//		case 'p':
	//			paused = !paused
	//			if paused {
	//				makeCall(client, stubs.Pause, world, convertParams(p))
	//				// Wait for another 'p' to unpause
	//				for {
	//					key = <-c.keyPresses
	//					if key == 'p' {
	//						makeCall(client, stubs.UnPause, world, convertParams(p))
	//						break
	//					}
	//				}
	//			} else {
	//				makeCall(client, stubs.UnPause, world, convertParams(p))
	//			}
	//		case 's':
	//			res, _ := makeCall(client, stubs.SendWorld, world, convertParams(p))
	//			world = res.World
	//			c.ioCommand <- ioOutput
	//			fileName := fmt.Sprintf("output_%d", res.Turn)
	//			c.ioFilename <- fileName
	//			for y := 0; y < p.ImageHeight; y++ {
	//				for x := 0; x < p.ImageWidth; x++ {
	//					c.ioOutput <- world[y][x]
	//				}
	//			}
	//			c.events <- ImageOutputComplete{res.Turn, fileName}
	//			fmt.Println("Saved current state to PGM image.")
	//
	//		case 'q':
	//			makeCall(client, stubs.StopServe, world, convertParams(p))
	//			close(c.events)
	//			os.Exit(0)
	//
	//		case 'k':
	//			makeCall(client, stubs.StopServe, world, convertParams(p))
	//			res, _ := makeCall(client, stubs.SendWorld, world, convertParams(p))
	//			world = res.World
	//			c.ioCommand <- ioOutput
	//			fileName := fmt.Sprintf("output_%d", res.Turn)
	//			c.ioFilename <- fileName
	//			for y := 0; y < p.ImageHeight; y++ {
	//				for x := 0; x < p.ImageWidth; x++ {
	//					c.ioOutput <- world[y][x]
	//				}
	//			}
	//			c.events <- ImageOutputComplete{res.Turn, fileName}
	//			fmt.Println("Saved current state to PGM image.")
	//		}
	//	}
	//}
	//
	//res, _ := makeCall(client, stubs.SendWorld, world, convertParams(p))
	//fileName := fmt.Sprintf("output_%d", res.Turn)
	//c.ioCommand <- ioOutput
	//c.ioFilename <- fileName
	//for y := 0; y < p.ImageHeight; y++ {
	//	for x := 0; x < p.ImageWidth; x++ {
	//		c.ioOutput <- world[y][x]
	//	}
	//}
	//
	//c.events <- ImageOutputComplete{res.Turn, fileName}
	//// Make sure that the Io has finished any output before exiting.
	//c.ioCommand <- ioCheckIdle
	//<-c.ioIdle
	//
	//c.events <- StateChange{turn, Quitting}
	//
	//// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	//close(c.events)
}
