package main

import (
	"flag"
	"fmt"
	"golDist/stubsClientToBroker"
	"golDist/stubsKeyPresses"
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


func (s *GameOfLife) ProcessAliveCellsCount(req stubsClientToBroker.RequestAliveCellsCount , res *stubsClientToBroker.ResponseToAliveCellsCount) (err error) {
	aliveCells := 0

	res.AliveCellsCount = aliveCells
	res.Turn = turn
	return
}

func (s *GameOfLife) ProcessWorld(req stubsClientToBroker.Request, res *stubsClientToBroker.Response) (err error) {

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
		if shutdown {
			time.Sleep(time.Second * 1)
			break
		}
	}
}
