package stubsServer

var ProcessWorldHandler = "GameOfLife.ProcessWorld"

//var keyPressesHandler = "GameOfLife.processKeyPresses"

type Response struct {
	ProcessedWorld [][]uint8
}

type Request struct {
	WorldSection [][]uint8
	ImageHeight  int
	ImageWidth   int
	Turns        int
}
