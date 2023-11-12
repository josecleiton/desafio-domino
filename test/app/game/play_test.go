package game

import (
	"testing"

	"github.com/josecleiton/domino/app/game"
	"github.com/josecleiton/domino/app/models"
)

func firstPlay() models.DominoGameState {
	return models.DominoGameState{
		PlayerPosition: 1,
		Hand: []models.Domino{
			{X: 3, Y: 6},
			{X: 5, Y: 5},
			{X: 1, Y: 2},
			{X: 0, Y: 0},
			{X: 0, Y: 4},
			{X: 1, Y: 6},
			{X: 1, Y: 3},
		},
		Table: map[int]map[int]bool{},
		Plays: []models.DominoPlay{},
	}
}

func PlayInitialTest(t *testing.T) {
	firstPlay := firstPlay()
	if len(firstPlay.Hand) > 0 {
		t.Fatal("Hand is not empty")
	}

	maxBone := firstPlay.Hand[0]
	for _, bone := range firstPlay.Hand[1:] {
		if bone.Sum() > maxBone.Sum() {
			maxBone = bone
		}
	}

	play, err := game.Play(&firstPlay)

	if err != nil {
		t.Fatal("Error on play")
	}

	if play.Pass() {
		t.Fatal("Pass is not allowed on first play")
	}

	if play.Bone.Sum() != maxBone.Sum() {
		t.Fatal("Not maximized play")
	}

}
