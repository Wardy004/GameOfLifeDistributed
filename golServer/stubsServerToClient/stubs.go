package stubsServerToClient

var ReverseHandler = "Client.HandleEvents"

type Response struct {
	//no reponse really needed
}

type Request struct {
	//create vars and completeworld slice for all events to be passed back to client
	AliveCellsCount int
	Image
}
