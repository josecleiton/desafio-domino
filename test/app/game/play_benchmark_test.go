package game

import (
	"fmt"
	"testing"

	"github.com/josecleiton/domino/app/game"
	"github.com/josecleiton/domino/app/models"
)

func BenchmarkTestPlayGlue(b *testing.B) {
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
					L: 5,
					R: 4,
				},
			},
		},
		{
			PlayerPosition: play.PlayerPosition + 2,
			Bone: models.DominoInTable{
				Edge: models.RightEdge,
				Domino: models.Domino{
					L: 5,
					R: 3,
				},
			},
		},
	}

	table := make(models.TableMap, len(plays))
	for _, play := range plays {
		if _, ok := table[play.Bone.L]; !ok {
			table[play.Bone.L] = make(models.TableBone, len(plays))
		}

		if _, ok := table[play.Bone.R]; !ok {
			table[play.Bone.R] = make(models.TableBone, len(plays))
		}

		table[play.Bone.L][play.Bone.R] = true
		table[play.Bone.R][play.Bone.L] = true
	}

	newHand := make([]models.Domino, 0, len(gameStateSt.Hand)-1)
	for _, bone := range gameStateSt.Hand {
		if bone != play.Bone.Domino {
			newHand = append(newHand, bone)
		}
	}

	gameStateNd := models.DominoGameState{
		PlayerPosition: play.PlayerPosition,
		Hand:           newHand,
		TableMap:       table,
		Plays:          plays,
	}

	ndPlay := game.Play(&gameStateNd)

	if ndPlay.Pass() {
		fmt.Println("Pass is not allowed")
		b.FailNow()
	}

	fromHand := false
	for _, bone := range newHand {
		if bone == ndPlay.Bone.Domino || bone.Reversed() == ndPlay.Bone.Domino {
			fromHand = true
		}
	}

	if !fromHand {
		fmt.Printf("Bone %v not found in hand %v\n", ndPlay.Bone.Domino, newHand)
		b.Fail()
	}

	fmt.Println(ndPlay)

}
