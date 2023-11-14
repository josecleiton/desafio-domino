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
		Table: models.Table{},
		Plays: []models.DominoPlay{},
		Edges: models.Edges{},
	}
}

func TestPlayInitial(t *testing.T) {
	firstPlay := firstPlay()
	if len(firstPlay.Hand) < 1 {
		t.Fatal("Hand is not empty")
	}

	play := game.Play(&firstPlay)

	if play.Pass() {
		t.Fatal("Pass is not allowed on first play")
	}

	maxBone := firstPlay.Hand[0]
	for _, bone := range firstPlay.Hand[1:] {
		if bone.Sum() > maxBone.Sum() {
			maxBone = bone
		}
	}

	if play.Bone.Sum() != maxBone.Sum() {
		t.Fatal("Not maximized play")
	}

}

func TestPlayGlue(t *testing.T) {
	gameStateSt := firstPlay()
	play := game.Play(&gameStateSt)

	plays := []models.DominoPlay{
		{
			PlayerPosition: play.PlayerPosition,
			Bone: models.DominoInTable{
				Edge:   play.Bone.Edge,
				Domino: play.Bone.Domino,
			},
		},
		{
			PlayerPosition: play.PlayerPosition + 1,
			Bone: models.DominoInTable{
				Edge: models.LeftEdge,
				Domino: models.Domino{
					X: 5,
					Y: 4,
				},
			},
		},
		{
			PlayerPosition: play.PlayerPosition + 2,
			Bone: models.DominoInTable{
				Edge: models.RightEdge,
				Domino: models.Domino{
					X: 5,
					Y: 3,
				},
			},
		},
	}

	table := make(models.Table, len(plays))
	for _, play := range plays {
		if _, ok := table[play.Bone.X]; !ok {
			table[play.Bone.X] = make(models.TableBone, len(plays))
		}

		if _, ok := table[play.Bone.Y]; !ok {
			table[play.Bone.Y] = make(models.TableBone, len(plays))
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

	edges := models.Edges{
		models.LeftEdge:  &plays[1],
		models.RightEdge: &plays[2],
	}

	gameStateNd := models.DominoGameState{
		PlayerPosition: play.PlayerPosition,
		Edges:          edges,
		Hand:           newHand,
		Table:          table,
		Plays:          plays,
	}

	play = game.Play(&gameStateNd)

	if play.Pass() {
		t.Fatal("Pass is not allowed on glue play")
	}
}
