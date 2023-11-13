package models

import "fmt"

const DominoLength = 28
const DominoUniqueBones = 7
const DominoHandLength = 7

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
	side := t.Side()
	return d.X == side || d.Y == side
}

func (t DominoInTable) Glue(d Domino) *DominoInTable {
	if !t.CanGlue(d) {
		return nil
	}

	return &DominoInTable{
		Domino: Domino{
			X: d.X,
			Y: d.Y,
		},
		Reversed: d.X == t.Side(),
	}
}

func (t DominoInTable) Side() int {
	if t.Reversed {
		return t.X
	}

	return t.Y
}
