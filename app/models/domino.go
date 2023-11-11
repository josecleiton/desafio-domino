package models

import "fmt"

type Domino struct {
	X, Y int
}

func (d Domino) String() string {
	return fmt.Sprintf("%d-%d", d.Y, d.X)
}

func DominoFromString(s string) Domino {
	var x, y int
	fmt.Sscanf(s, "%d-%d", &x, &y)
	return Domino{
		X: x,
		Y: y,
	}
}
