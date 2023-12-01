package models

import (
	"fmt"
)

const DominoMaxPlayer = 4
const DominoMinPlayer = 1
const DominoMaxEdges = 2

type Edge string
type PlayerPosition int

const (
	LeftEdge  Edge = "left"
	RightEdge Edge = "right"
	NoEdge    Edge = ""
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
	Plays          []DominoPlay
}

func (s DominoGameState) Edges() Edges {
	return Edges{
		LeftEdge:  &s.Table[0],
		RightEdge: &s.Table[len(s.Table)-1],
	}
}

func (play DominoPlayWithPass) Pass() bool {
	return play.Bone == nil
}

func (play DominoPlayWithPass) String() string {
	return fmt.Sprintf(
		"{Player: %d, Bone: %v}",
		play.PlayerPosition,
		play.Bone,
	)
}

func (d DominoInTable) GlueableSide() int {
	if d.Edge == LeftEdge {
		return d.L
	}

	return d.R
}

func (d DominoInTable) Reversed() DominoInTable {
	return DominoInTable{
		Edge:   d.Edge,
		Domino: d.Domino.Reversed(),
	}
}

func (d DominoInTable) Glue(other Domino) *Domino {
	side := d.GlueableSide()
	reversed := other.Reversed()

	if side == other.L {
		if d.Edge == LeftEdge {
			return &reversed
		}
		return &other
	}

	if side == other.R {
		if d.Edge == LeftEdge {
			return &other
		}
		return &reversed
	}

	return nil
}

func (p PlayerPosition) Add(count int) PlayerPosition {
	return PlayerPosition((int(p)+DominoMaxPlayer-1+count)%DominoMaxPlayer + 1)
}

func (p PlayerPosition) Next() PlayerPosition {
	return p.Add(1)
}

func (p PlayerPosition) Prev() PlayerPosition {
	return p.Add(-1)
}

func (table UnavailableBonesPlayer) Copy() UnavailableBonesPlayer {
	newTable := make(UnavailableBonesPlayer, DominoMaxPlayer)
	for i := DominoMinPlayer; i <= DominoMaxPlayer; i++ {
		newTable[PlayerPosition(i)] = make(TableBone, DominoUniqueBones)
	}

	for player, ub := range table {

		for k, v := range ub {
			if !v {
				continue
			}

			newTable[player][k] = true
		}
	}

	return newTable
}
