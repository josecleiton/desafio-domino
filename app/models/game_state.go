package models

type DominoPlay struct {
	PlayerPosition int
	Bone           Domino
	Reversed       bool
}

type DominoPlayWithPass struct {
	PlayerPosition int
	Bone           *Domino
	Reversed       bool
}

func (play DominoPlayWithPass) Pass() bool {
	return play.Bone == nil
}

const DominoLength = 28
const DominoUniqueBones = 7

type DominoGameState struct {
	PlayerPosition int
	Hand           []Domino
	Table          map[int]map[int]bool
	Plays          []DominoPlay
}
