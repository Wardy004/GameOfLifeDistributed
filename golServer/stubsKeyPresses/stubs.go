package stubsClient

var keyPressesHandler = "GameOfLifeKeyPress.HandleEvents"

type ResponseToKeyPress struct {
	//no reponse needed i think
}

type RequestFromKeyPress struct {
	//create vars and completeworld slice for all events to be passed back to client
	s bool
	q bool
	p bool
}
