package models

import "fmt"

const DominoMaxPlayer = 4
const DominoMinPlayer = 1

type Edge string
type PlayerPosition int

const (
	LeftEdge  Edge = "left"
	RightEdge Edge = "right"
)

type DominoPlay struct {
	PlayerPosition PlayerPosition
	Bone           DominoInTable
}

type DominoPlayWithPass struct {
	PlayerPosition PlayerPosition
	Bone           *DominoInTable
}

type Edges map[Edge]*Domino
type TableBone map[int]bool
type TableMap map[int]TableBone
type UnavailableBonesPlayer map[PlayerPosition]TableBone

type DominoGameState struct {
	PlayerPosition PlayerPosition
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

func (p PlayerPosition) Next() PlayerPosition {
	return PlayerPosition((int(p)-1)%DominoMaxPlayer + 1)
}

func (p PlayerPosition) Add(count int) PlayerPosition {
	return PlayerPosition((int(p)+count-1)%DominoMaxPlayer + 1)
}
