package models

import "fmt"

const DominoLength = 28
const DominoUniqueBones = 7
const DominoPlayerLength = 4

type Domino struct {
	X, Y int
}

func (d Domino) String() string {
	return fmt.Sprintf("%d-%d", d.Y, d.X)
}

func (d Domino) Sum() int {
	return d.X + d.Y
}

func DominoFromString(s string) Domino {
	var x, y int
	fmt.Sscanf(s, "%d-%d", &x, &y)
	return Domino{
		X: x,
		Y: y,
	}
}

func (d Domino) CanGlue(other Domino) bool {
	return d.X == other.X || d.X == other.Y || d.Y == other.X || d.Y == other.Y
}
