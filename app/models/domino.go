package models

import "fmt"

const DominoLength = 28
const DominoUniqueBones = 7
const DominoPlayerLength = 4

type Domino struct {
	X, Y int
}

type DominoInTable struct {
	Reversed bool
	Domino
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

func (t DominoInTable) CanGlue(d Domino) bool {
	side := t.X
	if t.Reversed {
		side = t.Y
	}

	return d.X == side || d.Y == side
}
