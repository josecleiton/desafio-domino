package models

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
type Table map[int]TableBone

type DominoGameState struct {
	PlayerPosition int
	Hand           []Domino
	Table          Table
	Edges          Edges
	Plays          []DominoPlay
}

func (play DominoPlayWithPass) Pass() bool {
	return play.Bone == nil
}

func (e Edges) Bones() []DominoInTable {
	bones := make([]DominoInTable, 0, len(e))
	for _, v := range e {
		bones = append(bones, v.Bone)
	}

	return bones
}
