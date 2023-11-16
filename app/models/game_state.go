package models

import "fmt"

const DominoMaxPlayer = 4
const DominoMinPlayer = 1

type Edge string

const (
	LeftEdge  Edge = "left"
	RightEdge Edge = "right"
)

type DominoPlay struct {
	PlayerPosition int
	Bone           DominoInTable
}

type DominoPlayWithPass struct {
	PlayerPosition int
	Bone           *DominoInTable
}

type Edges map[Edge]*Domino
type TableBone map[int]bool
type TableMap map[int]TableBone

type DominoGameState struct {
	PlayerPosition int
	Hand           []Domino
	Table          []Domino
	TableMap       TableMap
	Edges          Edges
	Plays          []DominoPlay
}

func (play DominoPlayWithPass) Pass() bool {
	return play.Bone == nil
}

func (play DominoPlayWithPass) String() string {
	return fmt.Sprintf("{Player: %d, Bone: %v}", play.PlayerPosition, play.Bone)
}

func (d DominoInTable) GlueableSide() int {
	if d.Edge == LeftEdge {
		return d.X
	}

	return d.Y
}

func (d DominoInTable) Glue(other Domino) *Domino {
	side := d.GlueableSide()

	if side == other.X {
		return &other
	}

	if side == other.Y {
		reversed := other.Reversed()
		return &reversed
	}

	return nil
}
