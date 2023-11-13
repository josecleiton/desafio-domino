package models

const DominoPlayerLength = 4

type DominoPlay struct {
	PlayerPosition int
	Bone           DominoInTable
}

func (play DominoPlay) CanGlue(d Domino) bool {
	return play.Bone.CanGlue(d)
}

type DominoPlayWithPass struct {
	PlayerPosition int
	Bone           *DominoInTable
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
