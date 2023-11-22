package stubs

import (
	"uk.ac.bris.cs/gameoflife/goUtil"
)

var LoadWorld = "Worker.LoadWorld"
var SendCellNumber = "Worker.SendCellNumber"
var SendWorld = "Worker.SendWorld"
var Pause = "Worker.Pause"
var UnPause = "Server.UnPause"
var StopServe = "Server.StopServe"
var ADD = "Worker.ADD"
var Load = "Worker.Load"

type Request struct {
	Params goUtil.Params
	World  goUtil.World
	Num    int
}

type Response struct {
	Turn    int
	World   goUtil.World
	Cellnum int
}
