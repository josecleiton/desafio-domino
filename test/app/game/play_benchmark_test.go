package game

import (
	"testing"

	"github.com/josecleiton/domino/app/game"
	"github.com/josecleiton/domino/app/models"
)

func BenchmarkTestPlayGlue(b *testing.B) {
	gameStateSt := firstPlay()
	play, err := game.Play(&gameStateSt)
	if err != nil {
		b.Fatal("Error on first play")
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
		b.Fatal("Error on play")
	}

	if play.Pass() {
		b.Fatal("Pass is not allowed on glue play")
	}

}
