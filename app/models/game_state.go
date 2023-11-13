package models

import "github.com/josecleiton/domino/app/utils"

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

type Plays map[Edge]*utils.LinkedList[DominoPlay]
type TableBone map[int]bool
type Table map[int]TableBone

type DominoGameState struct {
	PlayerPosition int
	Hand           []Domino
	Table          Table
	Plays          Plays
}

func (play DominoPlayWithPass) Pass() bool {
	return play.Bone == nil
}
