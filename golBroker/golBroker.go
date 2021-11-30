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

func makeWorkerSlice(world [][]uint8, blockLen,blockNo int) [][]uint8 {
	worldSection := makeMatrix(blockLen+2, ImageWidth)
	//top halo
	worldSection = append(worldSection,world[((blockNo*blockLen)-1+ImageHeight) % ImageHeight])
	//main section to be modified
	for x:=blockNo*blockLen;x<blockLen;x++{
		worldSection = append(worldSection,world[x])
	}
	//bottom halo
	worldSection = append(worldSection,world[((blockNo*(blockLen+1))+ImageHeight) % ImageHeight])
	return worldSection
}

func runWorker(WorkerSocket,BottomSocket string,section [][]uint8,blockLen,turns int, finishedSection chan<- [][]uint8) {
	fmt.Println("Worker: " + WorkerSocket)
	client, err := rpc.Dial("tcp", WorkerSocket)
	workers = append(workers, worker{client: client,ImageHeight:blockLen+2,ImageWidth: len(section[0])})
	if err != nil {panic(err)}
	defer client.Close()
	response := new(stubsBrokerToWorker.Response)
	//ImageHeight passed includes the halos

	// CHECKING SECTION HAS 255 VALUES IN IT
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if section[y][x] == 255{
				fmt.Println("Broker found a 1 to give to worker!")
			}
		}
	}

	request := stubsBrokerToWorker.Request{WorldSection:section,ImageHeight:blockLen+2,ImageWidth:len(section[0]) ,Turns: turns,BottomSocketAddress: BottomSocket}
	err = client.Call(stubsBrokerToWorker.ProcessWorldHandler, request, response)
	if err != nil {panic(err)}
	finishedSection <- section
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
	for _,worker := range workers {
		response := new(stubsClientToBroker.ResponseToAliveCellsCount)
		request := stubsClientToBroker.RequestAliveCellsCount{ImageHeight:worker.ImageHeight, ImageWidth:worker.ImageWidth}
		worker.client.Call(stubsClientToBroker.ProcessTimerEventsHandler,request,response)
		totalAliveCells += response.AliveCellsCount
	}
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

	if workers > 0 && workers <= req.ImageHeight  {
		for yPos := 0; yPos <= req.ImageHeight-blockLen; yPos += blockLen {
			BottomSocket := workerAddresses[(blockCount+workers+1)%workers]
			worldSection := makeWorkerSlice(req.WorldSection,blockLen,blockCount)
			outChannels = append(outChannels, make(chan [][]uint8))
			go runWorker(workerAddresses[blockCount],BottomSocket,worldSection,blockLen,req.Turns,outChannels[blockCount])
			blockCount++
			if blockCount == workers-1 && req.ImageHeight-(yPos+blockLen) > blockLen {break}
		}
		if blockCount != workers {
			BottomSocket := workerAddresses[0]
			worldSection := makeWorkerSlice(req.WorldSection,blockLen,blockCount)
			outChannels = append(outChannels, make(chan [][]uint8))
			go runWorker(workerAddresses[blockCount],BottomSocket,worldSection,blockLen,req.Turns,outChannels[blockCount])
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
