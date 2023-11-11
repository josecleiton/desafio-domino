package game

import (
	"sort"

	"github.com/josecleiton/domino/app/models"
)

var hand []models.Domino
var plays []models.DominoPlayWithPass
var states []models.DominoGameState

func initialize(state *models.DominoGameState) models.DominoPlayWithPass {
	hand = append(hand, state.Hand...)

	sort.Slice(hand, func(i, j int) bool {
		return hand[i].X+hand[i].Y >= hand[j].X+hand[j].Y
	})

	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone: &models.Domino{
			X: hand[0].X,
			Y: hand[0].Y,
		},
		Reversed: false,
	}
}

func Play(state *models.DominoGameState) models.DominoPlayWithPass {
	states = append(states, *state)

	if len(state.Plays) > 0 {
		// TODO: implementar lógica de jogo
		plays = append(plays, models.DominoPlayWithPass{
			PlayerPosition: state.PlayerPosition,
			Bone: &models.Domino{
				X: 6,
				Y: 6,
			},
			Reversed: false,
		})
	} else {
		plays = append(plays, initialize(state))
	}

	return plays[len(plays)-1]
}
