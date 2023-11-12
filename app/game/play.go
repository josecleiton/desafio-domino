package game

import (
	"crypto/rand"
	"log"
	"math/big"
	mrand "math/rand"
	"sort"

	"github.com/josecleiton/domino/app/models"
)

type nodeDomino struct {
	Left  models.Domino
	Right models.Domino
}

func (t *nodeDomino) Dominoes() []models.Domino {
	return []models.Domino{
		t.Left,
		t.Right,
	}
}

var hand []models.Domino
var plays []models.DominoPlayWithPass
var states []models.DominoGameState
var otherPlayersHasNot map[int]map[int]bool
var player int
var node *nodeDomino

func initialize(state *models.DominoGameState) models.DominoPlayWithPass {
	player = state.PlayerPosition
	clear(plays)
	clear(states)
	node = nil

	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone: &models.Domino{
			X: hand[0].X,
			Y: hand[0].Y,
		},
		Reversed: cryptoRandSecure(1024)&1 == 0,
	}
}

func cryptoRandSecure(max int64) int64 {
	nBig, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		log.Printf("Less secure random. Cause :%s\n", err)
		return mrand.Int63()
	}
	return nBig.Int64()
}

func Play(state *models.DominoGameState) models.DominoPlayWithPass {
	states = append(states, *state)
	hand = append(hand, state.Hand...)
	sort.Slice(hand, func(i, j int) bool {
		return hand[i].X+hand[i].Y >= hand[j].X+hand[j].Y
	})

	if len(state.Plays) > 0 {
		intermediateStates(state)
		onGoingGameplay := onGoingPlay(state)

		plays = append(plays, onGoingGameplay)
	} else {
		plays = append(plays, initialize(state))
	}

	return plays[len(plays)-1]
}

func onGoingPlay(state *models.DominoGameState) models.DominoPlayWithPass {
	// TODO: implementar lógica de jogo
	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone: &models.Domino{
			X: 6,
			Y: 6,
		},
		Reversed: false,
	}
}

func intermediateStates(state *models.DominoGameState) {
	var lastState *models.DominoGameState
	if len(states) > 0 {
		lastState = &states[len(states)-1]
	}

	currentPlay := state.Plays[len(state.Plays)-1]

	// update node with both edges of the domino map
	if node != nil && currentPlay.Bone != node.Left && currentPlay.Bone != node.Right {
		if currentPlay.Bone.CanGlue(node.Right) && currentPlay.Reversed {
			node.Right = currentPlay.Bone
		} else {
			node.Left = currentPlay.Bone
		}
	}

	// check if player passed and store it
	if node != nil && lastState.Plays[len(lastState.Plays)-1] == currentPlay {
		currentPlayerIdx := currentPlay.PlayerPosition - 1
		playerIdx := player - 1
		for i := currentPlayerIdx - 1; i != playerIdx; i = (i + 1) % currentPlayerIdx {
			playerPassed := i + 1

			for _, bone := range node.Dominoes() {
				otherPlayersHasNot[playerPassed][bone.X] = true
				otherPlayersHasNot[playerPassed][bone.Y] = true
			}
		}
	}

	canDetermineLeafs := len(state.Plays) > 2
	if canDetermineLeafs && node == nil {
		node = determineLeafs(state)
	}
}

func determineLeafs(state *models.DominoGameState) *nodeDomino {
	firstPlay := state.Plays[0]
	secondPlay := state.Plays[1]
	thirdPlay := state.Plays[2]

	thirdCanGlueSt := thirdPlay.Bone.CanGlue(firstPlay.Bone)
	thirdCanGlueNd := thirdPlay.Bone.CanGlue(secondPlay.Bone)
	if thirdCanGlueSt && !thirdCanGlueNd {
		return &nodeDomino{
			Left:  secondPlay.Bone,
			Right: thirdPlay.Bone,
		}
	} else {
		return &nodeDomino{
			Left:  firstPlay.Bone,
			Right: thirdPlay.Bone,
		}
	}

}
