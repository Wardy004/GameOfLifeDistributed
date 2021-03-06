package main

import (
	"flag"
	"fmt"
	"golDistributed/stubsServer"
	"math/rand"
	"net"
	"net/rpc"
	"time"
)

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

type GameOfLife struct{}

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

func (s *GameOfLife) ProcessWorld(req stubsServer.Request, res *stubsServer.Response) (err error) {
	turn := 0
	oWorld := makeMatrix(req.ImageHeight, req.ImageWidth)
	cpyWorld := makeMatrix(req.ImageHeight, req.ImageWidth)
	fmt.Println()
	for y := 0; y < req.ImageHeight; y++ {
		for x := 0; x < req.ImageWidth; x++ {
			oWorld[y][x] = req.WorldSection[y][x]
			cpyWorld[y][x] = oWorld[y][x]
			//if oWorld[y][x] == 255 {
			//c.events <- CellFlipped{CompletedTurns: 1, Cell: util.Cell{X: x, Y: y}}
			//}
		}
	}
	fmt.Println("Required turns: %T",req.Turns)
	for turn < req.Turns {
		turn++
		immutableWorld := makeImmutableMatrix(oWorld)
		performTurn(immutableWorld, cpyWorld, req.ImageHeight, req.ImageWidth)
		oWorld = cpyWorld
		cpyWorld = copySlice(oWorld)
		fmt.Println("Turn: %T", turn)
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
	rpc.Accept(listener)
}
