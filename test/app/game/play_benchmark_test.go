package game

import (
	"testing"

	"github.com/josecleiton/domino/app/game"
	"github.com/josecleiton/domino/app/models"
	"github.com/josecleiton/domino/app/utils"
)

func BenchmarkTestPlayGlue(b *testing.B) {
	gameStateSt := firstPlay()
	play := game.Play(&gameStateSt)

	leftEdge, rightEdge :=
		utils.LinkedList[models.DominoPlay]{},
		utils.LinkedList[models.DominoPlay]{}

	leftEdge.Push(&models.DominoPlay{
		PlayerPosition: play.PlayerPosition + 1,
		Bone: models.DominoInTable{
			Edge: models.LeftEdge,
			Domino: models.Domino{
				X: 5,
				Y: 4,
			},
		},
	})
	rightEdge.Push(&models.DominoPlay{
		PlayerPosition: play.PlayerPosition + 2,
		Bone: models.DominoInTable{
			Edge: models.RightEdge,
			Domino: models.Domino{
				X: 5,
				Y: 3,
			},
		},
	})
	plays := models.Plays{
		models.LeftEdge:  &leftEdge,
		models.RightEdge: &rightEdge,
	}

	table := make(models.Table, len(plays))
	for _, v := range plays {
		head := v.Head()
		for current := &head; current != nil; current = current.Next {
			bone := current.Data.Bone
			for _, boneSide := range []int{bone.X, bone.Y} {
				if _, ok := table[boneSide]; !ok {
					table[boneSide] = make(models.TableBone, len(plays))
				}
			}

			table[bone.X][bone.Y] = true
			table[bone.Y][bone.X] = true
		}
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

	play = game.Play(&gameStateNd)

	if play.Pass() {
		b.Fatal("Pass is not allowed on glue play")
	}

}
