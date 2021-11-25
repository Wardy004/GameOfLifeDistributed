package stubsWorkerToBroker

var HandleWorker = "GameOfLife.HandleWorker"

type Response struct {
	TopSocketAddress string
	BottomSocketAddress string
}

type Request struct {
	SocketAddress string
}
