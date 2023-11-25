package stubs

var CreateChannel = "Broker.CreateChannel"
var Publish = "Broker.Publish"
var Subscribe = "Broker.Subscribe"

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}

type WorldPart struct {
	StartRow int
	EndRow   int
	Data     [][]uint8
}

type Data struct {
	P     Params
	Turn  int
	World WorldPart
}

type PublishRequest struct {
	Method    string
	WorldData Data
}

type ChannelRequest struct {
	Method string
	Buffer int
}

type Subscription struct {
	Method        string
	ServerAddress string
	Callback      string
}

type JobReport struct {
	World [][]uint8
	Turn  int
}

type StatusReport struct {
	Message string
}
