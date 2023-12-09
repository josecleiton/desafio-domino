package models

import "fmt"

const DominoLength = 28
const DominoUniqueBones = 7
const DominoHandLength = 7
const DominoMaxBone = 6
const DominoMinBone = 0

type Domino struct {
	L, R int
}

type DominoInTable struct {
	Edge Edge
	Domino
}

func (d Domino) Sum() int {
	return d.L + d.R
}

func (d Domino) IsDouble() bool {
	return d.L == d.R
}

func (d Domino) Reversed() Domino {
	return Domino{L: d.R, R: d.L}
}

func (d Domino) Equals(other Domino) bool {
	if d.L^other.L == 0 && d.R^other.R == 0 {
		return true
	}

	reversed := other.Reversed()
	if d.L^reversed.L == 0 && d.R^reversed.R == 0 {
		return true
	}

	return false
}

func DominoFromString(s string) (*Domino, error) {
	var a, b int

	fmt.Sscanf(s, "%d-%d", &a, &b)

	if a < DominoMinBone || a > DominoMaxBone {
		return nil, fmt.Errorf("invalid bone: %d", a)
	}
	if b < DominoMinBone || b > DominoMaxBone {
		return nil, fmt.Errorf("invalid bone: %d", b)
	}

	return &Domino{
		L: a,
		R: b,
	}, nil
}
