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

type DominoGameState struct {
	PlayerPosition int
	Hand           []Domino
	Table          map[int]map[int]bool
	Plays          []DominoPlay
}
