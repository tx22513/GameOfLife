package stubs

import (
	"uk.ac.bris.cs/gameoflife/goUtils"
)

type Request struct {
	Params goUtils.Params
	World  [][]uint8
}

type Response struct {
	Turn    int
	World   [][]uint8
	Cellnum int
	Message string
}

var LoadWorld = "Server.LoadWorld"
var Update = "Server.Update"
var SendCellNumber = "Server.SendCellNumber"
var SendCurrentState = "Server.SendCurrentState"
var Pause = "Server.Pause"
var UnPause = "Server.UnPause"
var DisconnectClient = "Server.DisconnectClient"
var ShotDown = "Server.ShotDown"
