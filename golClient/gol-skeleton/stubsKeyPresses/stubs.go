package stubsKeyPresses

var KeyPressesHandler = "GameOfLifeKeyPress.HandleEvents"

type ResponseToKeyPress struct {
	//case s world is returned
	WorldSection [][]uint8
}

type RequestFromKeyPress struct {
	KeyPressed string
}
