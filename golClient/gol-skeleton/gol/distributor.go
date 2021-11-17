package gol

import "C"
import (
	"fmt"
	"net/rpc"

	"uk.ac.bris.cs/gameoflife/stubsServer"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

func makeMatrix(height, width int) [][]uint8 {
	matrix := make([][]uint8, height)
	for i := range matrix {
		matrix[i] = make([]uint8, width)
	}
	return matrix
}

func outputBoard(world [][]byte, p Params, c distributorChannels) {
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, p.Turns)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}
}

func processKeyPresses(c distributorChannels, keyPresses <-chan rune) {
	for {
		select {
		case key := <-keyPresses:
			switch key {
			case 112: // P: pause processing and print current turn, if p pressed again then resume

			case 113: // Q: Generate PGM file with current state of board and terminate

			case 115: // S: Generate PGM file with current state of board

			}
		}
	}
}

func getLiveCells(p Params, oWorld [][]uint8) []util.Cell {
	liveCells := make([]util.Cell, 0)
	for x := 0; x < p.ImageWidth; x++ {
		for y := 0; y < p.ImageHeight; y++ {
			if oWorld[y][x] == 255 {
				var currentCell = util.Cell{X: x, Y: y}
				liveCells = append(liveCells, currentCell)
			}
		}
	}
	return liveCells
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%vx%v", p.ImageWidth, p.ImageHeight)
	oWorld := makeMatrix(p.ImageHeight, p.ImageWidth)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			oWorld[y][x] = <-c.ioInput
		}
	}
	server := "127.0.0.1:8030"
	fmt.Println("Server: " + server)
	client, err := rpc.Dial("tcp", server)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer client.Close()

	go processKeyPresses(c, keyPresses)

	outputBoard(oWorld, p, c)

	response := new(stubsServer.Response)
	request := stubsServer.Request{WorldSection: oWorld, ImageHeight: p.ImageHeight, ImageWidth: p.ImageWidth, Turns: p.Turns}
	client.Call(stubsServer.ProcessWorldHandler, request, response)

	cellsAlive := getLiveCells(p, response.ProcessedWorld)
	// Make sure that the Io has finished any output before exiting.
	c.events <- FinalTurnComplete{CompletedTurns: p.Turns, Alive: cellsAlive}
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle
	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}
