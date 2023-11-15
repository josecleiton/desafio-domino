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

type Edges map[Edge]*DominoPlay
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

func (e Edges) Bones() []DominoInTable {
	bones := make([]DominoInTable, 0, len(e))
	for _, v := range e {
		bones = append(bones, v.Bone)
	}

	return bones
}
func (d DominoInTable) Glue(other Domino) *Domino {
	side := d.X
	if d.Edge == RightEdge {
		side = d.Y
	}

	if side == other.X {
		return &other
	}

	if side == other.Y {
		reversed := other.Reversed()
		return &reversed
	}

	return nil
}
