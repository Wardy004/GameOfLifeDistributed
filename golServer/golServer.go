package main

import (
	"flag"
	"fmt"
	"golDistributed/stubsClientToServer"
	"golDistributed/stubsKeyPresses"
	"math/rand"
	"net"
	"net/rpc"
	"time"
)

var oWorld [][]uint8
var turn int
var pause,quit chan bool
var shutdown bool
type GameOfLife struct{}


func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

func makeImmutableMatrix(matrix [][]uint8) func(y, x int) uint8 {
	return func(y, x int) uint8 {
		return matrix[y][x]
	}
}

func performTurn(world func(y, x int) uint8, newWorld [][]uint8, imageHeight, imageWidth int) {

	for y := 0; y < imageHeight; y++ {
		for x := 0; x < imageWidth; x++ {

			aliveCells := 0
			//top left
			aliveCells += int(world((y+imageHeight-1)%imageHeight, (x+imageWidth-1)%imageWidth))
			//top middle
			aliveCells += int(world((y+imageHeight-1)%imageHeight, x))
			//top right
			aliveCells += int(world((y+imageHeight-1)%imageHeight, (x+imageWidth+1)%(imageWidth)))
			//middle left
			aliveCells += int(world(y, (x+imageWidth-1)%imageWidth))
			//middle right
			aliveCells += int(world(y, (x+imageWidth+1)%imageWidth))
			//bottom left
			aliveCells += int(world((y+imageHeight+1)%imageHeight, (x+imageWidth-1)%imageWidth))
			//bottom middle
			aliveCells += int(world((y+imageHeight+1)%imageHeight, x))
			//bottom right
			aliveCells += int(world((y+imageHeight+1)%imageHeight, (x+imageWidth+1)%imageWidth))
			if aliveCells > 0 {
				aliveCells = aliveCells / 255

			}
			if world(y, x) == 255 {
				if aliveCells < 2 || aliveCells > 3 {
					newWorld[y][x] = 0
					//c.events <- CellFlipped{CompletedTurns: 1, Cell: util.Cell{X: x, Y: y}}
				}
			} else {
				if aliveCells == 3 {
					newWorld[y][x] = 255
					//c.events <- CellFlipped{CompletedTurns: 1, Cell: util.Cell{X: x, Y: y}}
				}
			}
		}
	}
}

func copySlice(original [][]uint8) [][]uint8 {
	height := len(original)
	width := len(original[0])
	sliceCopy := makeMatrix(height, width)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sliceCopy[y][x] = original[y][x]
		}
	}
	return sliceCopy
}

func (s *GameOfLife) ProcessKeyPresses(req stubsKeyPresses.RequestFromKeyPress, res *stubsKeyPresses.ResponseToKeyPress) (err error) {
		switch req.KeyPressed {
		case "p":
			res.Turn = turn
			pause<-true
			fmt.Println(fmt.Sprintf("Puased on turn: %d",turn))
		case "q":
			fmt.Println("q pressed")
			quit<-true
		case "s":
			fmt.Println("s pressed")
			res.WorldSection = oWorld
		case "k":
			fmt.Println("k pressed")
			quit<-true
			shutdown=true
		}
	return
}


func (s *GameOfLife) ProcessAliveCellsCount(req stubsClientToServer.RequestAliveCellsCount , res *stubsClientToServer.ResponseToAliveCellsCount) (err error) {
	aliveCells := 0
	for y := 0; y < req.ImageHeight; y++ {
		for x := 0; x < req.ImageWidth; x++ {
			if oWorld[y][x] == 255 {
				aliveCells++
			}
		}
	}
	res.AliveCellsCount = aliveCells
	res.Turn = turn
	return
}

func (s *GameOfLife) ProcessWorld(req stubsClientToServer.Request, res *stubsClientToServer.Response) (err error) {
	turn = 0
	quit = make(chan bool)
	pause = make(chan bool)
	oWorld = makeMatrix(req.ImageHeight, req.ImageWidth)
	cpyWorld := makeMatrix(req.ImageHeight, req.ImageWidth)
	
	for y := 0; y < req.ImageHeight; y++ {
		for x := 0; x < req.ImageWidth; x++ {
			oWorld[y][x] = req.WorldSection[y][x]
			cpyWorld[y][x] = oWorld[y][x]
			//if oWorld[y][x] == 255 {
			//c.events <- CellFlipped{CompletedTurns: 1, Cell: util.Cell{X: x, Y: y}}
			//}
		}
	}
	quit:
	for turn < req.Turns {
		fmt.Println(fmt.Sprintf("Turn: %d",turn))
		select {
		case <-pause:
			<-pause
			fmt.Println("Resumed")
		case <-quit:
			break quit
		default:
			immutableWorld := makeImmutableMatrix(oWorld)
			performTurn(immutableWorld, cpyWorld, req.ImageHeight, req.ImageWidth)
			turn++
			oWorld = cpyWorld
			cpyWorld = copySlice(oWorld)
		}
	}
	res.ProcessedWorld = oWorld
	return
}

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	rpc.Register(&GameOfLife{})
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	go rpc.Accept(listener)
	for {
		if shutdown {break}
	}
}
