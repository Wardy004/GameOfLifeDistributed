package main

import (
	"flag"
	"fmt"
	"golDistributed/stubsBrokerToWorker"
	"golDistributed/stubsKeyPresses"
	"golDistributed/stubsWorkerToBroker"
	"golDistributed/stubsWorkerToWorker"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"
)

var oWorld [][]uint8
var Turn int
var Pause chan bool
var Quit chan bool
var RowExchange chan bool
var Shutdown bool

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

	for y := 1; y < imageHeight-1; y++ { //from 1 to <= to account for padding
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

func getBottomHalo(BottomWorker *rpc.Client) {
	request := stubsWorkerToWorker.RequestRow{Turn: Turn,Row: oWorld[len(oWorld)-1]} //pass bottom row to bottom worker
	response := new(stubsWorkerToWorker.ResponseRow)
	BottomWorker.Call(stubsWorkerToWorker.ProcessRowExchange,request,response) //get bottom row from bottom worker
	oWorld[len(oWorld)-1] = response.Row
	RowExchange<-true
}

func (s *GameOfLife) ProcessKeyPresses(req stubsKeyPresses.RequestFromKeyPress, res *stubsKeyPresses.ResponseToKeyPress) (err error) {
		switch req.KeyPressed {
		case "p":
			res.Turn = Turn
			Pause<-true
			fmt.Println(fmt.Sprintf("Puased on Turn: %d",Turn))
		case "q":
			fmt.Println("q pressed")
			Quit<-true
		case "s":
			fmt.Println("s pressed")
			res.WorldSection = oWorld
		case "k":
			fmt.Println("k pressed")
			Quit<-true
			Shutdown=true
		}
	return
}

func (s *GameOfLife) ProcessAliveCellsCount(req stubsBrokerToWorker.RequestAliveCellsCount , res *stubsBrokerToWorker.ResponseToAliveCellsCount) (err error) {
	aliveCells := 0
	for y := 1; y < req.ImageHeight-1; y++ { //Halo regions avoided
		for x := 0; x < req.ImageWidth; x++ {
			if oWorld[y][x] == 255 {
				aliveCells++
			}
		}
	}
	res.AliveCellsCount = aliveCells
	res.Turn = Turn
	return
}

func (s *GameOfLife) ProcessRowExchange(req stubsWorkerToWorker.RequestRow , res *stubsWorkerToWorker.ResponseRow) (err error) {
	for {
		if req.Turn == Turn {
			oWorld[0] = req.Row
			res.Row = oWorld[1]
			break
		}
	}
	RowExchange<-true
	return
}

func (s *GameOfLife) ProcessWorld(req stubsBrokerToWorker.Request, res *stubsBrokerToWorker.Response) (err error) {
	fmt.Println("I'm a worker about to process a world")
	Turn = 0
	Quit = make(chan bool)
	Pause = make(chan bool)
	RowExchange = make(chan bool)
	fmt.Println("ProcessWorld 1")
	BottomWorker, err := rpc.Dial("tcp",req.BottomSocketAddress)
	fmt.Println("ProcessWorld 2")
	oWorld = makeMatrix(req.ImageHeight, req.ImageWidth)
	cpyWorld := makeMatrix(req.ImageHeight, req.ImageWidth)
	fmt.Println("section height is: ", req.ImageHeight)
	fmt.Println("section width is: ", req.ImageWidth)

	for y := 0; y < req.ImageHeight; y++ {
		for x := 0; x < req.ImageWidth; x++ {
			//fmt.Println("ProcessWorld 2.1")
			oWorld[y][x] = req.WorldSection[y][x]
			cpyWorld[y][x] = oWorld[y][x]
		}
	}
	fmt.Println("Number of turns is", req.Turns)

	//Quit:
	for Turn < req.Turns {
		fmt.Println(fmt.Sprintf("Turn: %d",Turn))
		select {
		case <-Pause:
			<-Pause
			fmt.Println("Resumed")
		case <-Quit:
			//break Quit
		default:
			fmt.Println("ProcessWorld 3")
			immutableWorld := makeImmutableMatrix(oWorld)
			fmt.Println("ProcessWorld 4")
			performTurn(immutableWorld, cpyWorld, req.ImageHeight, req.ImageWidth)
			fmt.Println("ProcessWorld 5")
			oWorld = cpyWorld
			Turn++
			fmt.Println("ProcessWorld 6")
			go getBottomHalo(BottomWorker)
			fmt.Println("ProcessWorld 7")
			<-RowExchange
			<-RowExchange
			fmt.Println("ProcessWorld 8")
			cpyWorld = copySlice(oWorld)
		}
	}
	res.ProcessedSection = oWorld
	return
}

func main() {
	mySocketAddress := os.Args[1]
	broker := os.Args[2]
	fmt.Println("Server: " + broker)
	fmt.Println("worker 1")
	client, err := rpc.Dial("tcp", broker)
	fmt.Println("worker 2")
	if err != nil {
		panic(err)
	}
	fmt.Println("worker 3")
	defer client.Close()
	fmt.Println("worker 4")
	response := new(stubsWorkerToBroker.Response)
	fmt.Println("worker 5")
	request := stubsWorkerToBroker.Request{SocketAddress: mySocketAddress}
	fmt.Println("worker 6")
	err = client.Call(stubsWorkerToBroker.HandleWorker, request, response)
	fmt.Println("worker 7")
	if err != nil {
		panic(err)
	}

	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	fmt.Println("worker 8")
	rpc.Register(&GameOfLife{})
	fmt.Println("worker 9")
	listener, _ := net.Listen("tcp", ":"+*pAddr)
	defer listener.Close()
	go rpc.Accept(listener)
	for {
		if Shutdown {
			time.Sleep(time.Second * 1)
			break
		}
	}
}
