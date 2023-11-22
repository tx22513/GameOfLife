package goUtil

type World [][]byte

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

func CreateNewWorld(height, width int) World {
	newWorld := make([][]uint8, height)
	for v := range newWorld {
		newWorld[v] = make([]uint8, width)
	}
	return newWorld
}
