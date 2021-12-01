package main

import (
	"flag"
	"fmt"
	"golDist/stubsBrokerToWorker"
	"golDist/stubsClientToBroker"
	"golDist/stubsKeyPresses"
	"golDist/stubsWorkerToBroker"
	"math"
	"math/rand"
	"net"
	"net/rpc"
	"time"
)

var shutdown bool
var workerAddresses []string
var workers []worker
var ImageHeight int
var ImageWidth int
type GameOfLife struct{}

type worker struct {
	client *rpc.Client
	ImageHeight int
	ImageWidth int
}

func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

func printWorld(world [][]uint8) {
	for y:=1;y<len(world)-1;y++{
		fmt.Println(world[y])
	}
}

func makeWorkerSlice(world [][]uint8, blockLen,blockNo int) [][]uint8 {
	worldSection := makeMatrix(blockLen+2, ImageWidth)
	for x:=blockLen*blockNo;x<blockNo*blockLen+blockLen+2;x++{
		worldSection[x-blockLen*blockNo] = world[(x-1+ImageHeight) % ImageHeight]
	}
	//fmt.Println("Worker slice: ")
	//printWorld(worldSection)
	return worldSection
}

func runWorker(WorkerSocket,BottomSocket string,section [][]uint8,blockLen,turns int, finishedSection chan<- [][]uint8, interrupt chan<- bool) {
	fmt.Println("Worker: " + WorkerSocket)
	client, err := rpc.Dial("tcp", WorkerSocket)
	workers = append(workers,worker{client: client,ImageHeight:blockLen+2,ImageWidth:len(section[0])})
	if err != nil {panic(err)}
	defer client.Close()
	response := new(stubsBrokerToWorker.Response)
	//ImageHeight passed includes the halos
	request := stubsBrokerToWorker.Request{WorldSection:section,ImageHeight:blockLen+2,ImageWidth:len(section[0]) ,Turns: turns,BottomSocketAddress: BottomSocket}
	err = client.Call(stubsBrokerToWorker.ProcessWorldHandler, request, response)
	if err != nil {panic(err)}
	finishedSection <- response.ProcessedSection
}

func (s *GameOfLife) RegisterWorker(req stubsWorkerToBroker.Request, res *stubsWorkerToBroker.Response) (err error) {
	fmt.Println("registering a worker")
	workerAddresses = append(workerAddresses, req.SocketAddress)
	res = nil
	return
}

func (s *GameOfLife) ProcessKeyPresses(req stubsKeyPresses.RequestFromKeyPress, res *stubsKeyPresses.ResponseToKeyPress) (err error) {
	var currentWorld [][]uint8
	if req.KeyPressed == "s" || req.KeyPressed == "k" {currentWorld = makeMatrix(ImageHeight,ImageWidth)}
	for _,worker := range workers {
		worker.client.Call(stubsKeyPresses.KeyPressHandler,req,res)
		if req.KeyPressed == "s" || req.KeyPressed == "k" {currentWorld = append(currentWorld, res.WorldSection...)}
	}
	if req.KeyPressed == "s" || req.KeyPressed == "k" {res.WorldSection = currentWorld}
	if req.KeyPressed == "k" {shutdown = true}
	return
}

func (s *GameOfLife) ProcessAliveCellsCount(req stubsClientToBroker.RequestAliveCellsCount , res *stubsClientToBroker.ResponseToAliveCellsCount) (err error) {
	totalAliveCells := 0
	turnA := 0
	turnB := 0
	for i,worker := range workers {
		response := new(stubsBrokerToWorker.ResponseToAliveCellsCount)
		request := stubsBrokerToWorker.RequestAliveCellsCount{ImageHeight:worker.ImageHeight, ImageWidth:worker.ImageWidth}
		worker.client.Call(stubsBrokerToWorker.ProcessTimerEventsHandler,request,response)
		totalAliveCells += response.AliveCellsCount
		if i == 0 { turnA = response.Turn}
		if i == 1 { turnB = response.Turn}
	}
	if turnA != turnB {fmt.Println("mismatched turns")}
	res.Turn = turnA
	fmt.Println("alive cells is", totalAliveCells, "at turn", res.Turn)
	res.AliveCellsCount = totalAliveCells
	return
}

func (s *GameOfLife) ProcessWorld(req stubsClientToBroker.Request, res *stubsClientToBroker.Response) (err error) {
	blockCount := 0
	ImageHeight = req.ImageHeight
	ImageWidth = req.ImageWidth
	workers := len(workerAddresses)
	blockLen := int(math.Floor(float64(req.ImageHeight) / float64(workers)))
	outChannels := make([]chan [][]uint8, 0)
	interrupt := make(chan bool)

	if workers > 0 && workers <= req.ImageHeight  {
		//printWorld(req.WorldSection)
		for yPos := 0; yPos <= req.ImageHeight-blockLen; yPos += blockLen {
			BottomSocket := workerAddresses[(blockCount+workers+1)%workers]
			worldSection := makeWorkerSlice(req.WorldSection,blockLen,blockCount)
			outChannels = append(outChannels, make(chan [][]uint8))
			go runWorker(workerAddresses[blockCount],BottomSocket,worldSection,blockLen,req.Turns,outChannels[blockCount], interrupt)
			blockCount++
			if blockCount == workers-1 && req.ImageHeight-(yPos+blockLen) > blockLen {break}
		}
		if blockCount != workers {
			BottomSocket := workerAddresses[0]
			worldSection := makeWorkerSlice(req.WorldSection,blockLen,blockCount)
			outChannels = append(outChannels, make(chan [][]uint8))
			go runWorker(workerAddresses[blockCount],BottomSocket,worldSection,blockLen,req.Turns,outChannels[blockCount], interrupt)
			blockCount++
		}
		finishedWorld := make([][]uint8, 0)
		for block := 0; block < workers; block++ {
			finishedWorld = append(finishedWorld, <-outChannels[block]...)
		}

		res.ProcessedWorld = finishedWorld
	} else {panic("No workers available")}

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
