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

func TestPlayInitial(t *testing.T) {
	firstPlay := firstPlay()
	if len(firstPlay.Hand) < 1 {
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

func TestPlayGlue(t *testing.T) {
	gameStateSt := firstPlay()
	play, err := game.Play(&gameStateSt)
	if err != nil {
		t.Fatal("Error on first play")
	}

	plays := []models.DominoPlay{
		{
			PlayerPosition: play.PlayerPosition,
			Bone: models.DominoInTable{
				Domino:   play.Bone.Domino,
				Reversed: play.Bone.Reversed,
			},
		},
		{
			PlayerPosition: play.PlayerPosition + 1,
			Bone: models.DominoInTable{
				Domino: models.Domino{
					X: 5,
					Y: 4,
				},
				Reversed: false,
			},
		},
		{
			PlayerPosition: play.PlayerPosition + 2,
			Bone: models.DominoInTable{
				Domino: models.Domino{
					X: 5,
					Y: 3,
				},
				Reversed: false,
			},
		},
	}

	table := make(map[int]map[int]bool, len(plays))
	for _, play := range plays {
		if _, ok := table[play.Bone.X]; !ok {
			table[play.Bone.X] = make(map[int]bool, len(plays))
		}
		if _, ok := table[play.Bone.Y]; !ok {
			table[play.Bone.Y] = make(map[int]bool, len(plays))
		}
		table[play.Bone.X][play.Bone.Y] = true
		table[play.Bone.Y][play.Bone.X] = true
	}

	newHand := make([]models.Domino, 0, len(gameStateSt.Hand)-1)
	for _, bone := range gameStateSt.Hand {
		if bone != play.Bone.Domino {
			newHand = append(newHand, bone)
		}
	}

	gameStateNd := models.DominoGameState{
		PlayerPosition: play.PlayerPosition,
		Plays:          plays,
		Hand:           newHand,
		Table:          table,
	}

	play, err = game.Play(&gameStateNd)
	if err != nil {
		t.Fatal("Error on play")
	}

	if play.Pass() {
		t.Fatal("Pass is not allowed on glue play")
	}
}
