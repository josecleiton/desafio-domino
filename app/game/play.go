package game

import (
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	mrand "math/rand"
	"sort"

	"github.com/josecleiton/domino/app/models"
)

type nodeDomino struct {
	Left  models.DominoInTable
	Right models.DominoInTable
}

func (t *nodeDomino) Dominoes() []models.Domino {
	return []models.Domino{t.Left.Domino, t.Right.Domino}
}

var hand []models.Domino
var plays []models.DominoPlayWithPass
var states []models.DominoGameState
var unavailableBones map[int]map[int]bool
var player int
var node *nodeDomino

func initialize(state *models.DominoGameState) models.DominoPlayWithPass {
	player = state.PlayerPosition
	clear(plays)
	clear(states)
	node = nil

	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone: &models.DominoInTable{
			Reversed: cryptoRandSecure(1024)&1 == 0,
			Domino: models.Domino{
				X: hand[0].X,
				Y: hand[0].Y,
			},
		},
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

func Play(state *models.DominoGameState) (*models.DominoPlayWithPass, error) {
	states = append(states, *state)
	hand = append(hand, state.Hand...)
	sort.Slice(hand, func(i, j int) bool {
		return hand[i].Sum() >= hand[j].Sum()
	})
	plays = make([]models.DominoPlayWithPass, 0, len(state.Plays))

	for _, play := range state.Plays {
		if play.PlayerPosition != player {
			continue
		}

		plays = append(plays, models.DominoPlayWithPass{
			PlayerPosition: player,
			Bone:           &play.Bone,
		})

	}

	if len(state.Plays) > 0 {
		intermediateStates(state)
		if player != state.PlayerPosition {
			return nil, errors.New("not my turn")
		}
		plays = append(plays, midgameDecision(state))
	} else {
		plays = append(plays, initialize(state))
	}

	return &plays[len(plays)-1], nil
}

func midgameDecision(state *models.DominoGameState) models.DominoPlayWithPass {
	var edges *nodeDomino
	if node != nil {
		edges = node
	}

	left, right := handCanPlayThisTurn(edges)
	leftLen, rightLen := len(left), len(right)

	// pass
	if leftLen == 0 && rightLen == 0 {
		for _, bone := range node.Dominoes() {
			unavailableBones[player][bone.X] = true
			unavailableBones[player][bone.Y] = true
		}

		return models.DominoPlayWithPass{PlayerPosition: player}
	}

	canPlayBoth := leftLen > 0 && rightLen > 0
	if !canPlayBoth {
		return oneSidedPlay(left, right)
	}

	// TODO: implementar decisão
	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone: &models.DominoInTable{
			Reversed: false,
			Domino:   hand[0],
		},
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
		if node.Right.CanGlue(currentPlay.Bone.Domino) {
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
				unavailableBones[playerPassed][bone.X] = true
				unavailableBones[playerPassed][bone.Y] = true
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

	thirdCanGlueSt := thirdPlay.CanGlue(firstPlay.Bone.Domino)
	thirdCanGlueNd := thirdPlay.CanGlue(secondPlay.Bone.Domino)
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

func handCanPlayThisTurn(edges *nodeDomino) ([]models.Domino, []models.Domino) {
	if edges == nil {
		return nil, nil
	}

	bonesGlueLeft := make([]models.Domino, 0, len(hand))
	bonesGlueRight := make([]models.Domino, 0, len(hand))
	for _, h := range hand {
		if node.Left.CanGlue(h) {
			bonesGlueLeft = append(bonesGlueLeft, models.Domino{
				X: h.X,
				Y: h.Y,
			})
		}

		if node.Right.CanGlue(h) {
			bonesGlueRight = append(bonesGlueRight, models.Domino{
				X: h.X,
				Y: h.Y,
			})
		}
	}

	return bonesGlueLeft, bonesGlueRight
}

func oneSidedPlay(left []models.Domino, right []models.Domino) models.DominoPlayWithPass {
	if len(left) > 0 {
		return maximizedPlay(left, node.Left)
	}

	return maximizedPlay(right, node.Right)
}

func maximizedPlay(bones []models.Domino, edge models.DominoInTable) models.DominoPlayWithPass {

	maxBone := bones[0]
	for _, bone := range bones[1:] {
		if bone.Sum() > maxBone.Sum() {
			maxBone = bone
		}
	}

	return playFromEdge(maxBone, edge)
}

func playFromEdge(bone models.Domino, edge models.DominoInTable) models.DominoPlayWithPass {
	return models.DominoPlayWithPass{
		PlayerPosition: player,
		Bone: &models.DominoInTable{
			Domino: models.Domino{
				X: bone.X,
				Y: bone.Y,
			},
			Reversed: bone.Y == edge.X,
		},
	}
}
