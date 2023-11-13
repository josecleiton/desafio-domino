package models

const DominoMaxPlayer = 4
const DominoMinPlayer = 1

type Edge string

const (
	Left  Edge = "left"
	Right Edge = "right"
)

type DominoPlay struct {
	PlayerPosition int
	Bone           DominoInTable
}

type DominoPlayWithPass struct {
	PlayerPosition int
	Bone           *DominoInTable
}

type DominoGameState struct {
	PlayerPosition int
	Hand           []Domino
	Table          map[int]map[int]bool
	Plays          []DominoPlay
}

func (play DominoPlay) CanGlue(d Domino) bool {
	return play.Bone.CanGlue(d)
}

func (play DominoPlayWithPass) Pass() bool {
	return play.Bone == nil
}
