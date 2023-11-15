package models

import "fmt"

const DominoLength = 28
const DominoUniqueBones = 7
const DominoHandLength = 7
const DominoMaxBone = 6
const DominoMinBone = 0

type Domino struct {
	X, Y int
}

type DominoInTable struct {
	Edge Edge
	Domino
}

func (d Domino) Sum() int {
	return d.X + d.Y
}

func (d Domino) IsDouble() bool {
	return d.X == d.Y
}

func (d Domino) Reversed() Domino {
	return Domino{X: d.Y, Y: d.X}
}

func DominoFromString(s string) (*Domino, error) {
	var x, y int

	fmt.Sscanf(s, "%d-%d", &x, &y)

	if x > DominoMaxBone || x < DominoMinBone {
		return nil, fmt.Errorf("invalid bone: %d", x)
	}
	if y > DominoMaxBone || y < DominoMinBone {
		return nil, fmt.Errorf("invalid bone: %d", y)
	}

	return &Domino{
		X: x,
		Y: y,
	}, nil
}
