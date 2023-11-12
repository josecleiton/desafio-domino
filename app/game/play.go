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
			Reversed:       play.Reversed,
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
	canPlayBoth := left != nil && right != nil
	if !canPlayBoth {
		return oneSidedPlay(left, right)
	}

	if node == nil {
		plays := [...]models.DominoPlayWithPass{
			playFromEdge(left, node.Left),
			playFromEdge(right, node.Right),
		}
		return maximizedPlay(plays[:]...)
	}

	// TODO: implementar decisão
	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone:           &hand[0],
		Reversed:       false,
	}
}

func maximizedPlay(plays ...models.DominoPlayWithPass) models.DominoPlayWithPass {
	sort.Slice(plays, func(i, j int) bool {
		return plays[i].Bone.Sum() >= plays[j].Bone.Sum()
	})

	return plays[0]
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

func handCanPlayThisTurn(edges *nodeDomino) (*models.Domino, *models.Domino) {
	if edges == nil {
		return nil, nil
	}

	var boneGlueLeft, boneGlueRight *models.Domino
	for _, h := range hand {
		if h.CanGlue(node.Left) {
			boneGlueLeft = &models.Domino{
				X: h.X,
				Y: h.Y,
			}
		}

		if h.CanGlue(node.Right) {
			boneGlueRight = &models.Domino{
				X: h.X,
				Y: h.Y,
			}
		}

		if boneGlueLeft != nil && boneGlueRight != nil {
			break
		}
	}

	hasCurrentBone := boneGlueLeft != nil || boneGlueRight != nil

	// passed
	if !hasCurrentBone {
		for _, bone := range [...]models.Domino{node.Left, node.Right} {
			unavailableBones[player][bone.X] = true
			unavailableBones[player][bone.Y] = true
		}

		return nil, nil
	}
	return boneGlueLeft, boneGlueRight
}

func oneSidedPlay(left *models.Domino, right *models.Domino) models.DominoPlayWithPass {
	if left != nil {
		return playFromEdge(left, node.Left)
	}

	return models.DominoPlayWithPass{
		PlayerPosition: player,
		Bone:           right,
		Reversed:       right.Y == node.Right.X,
	}
}

func playFromEdge(bone *models.Domino, edge models.Domino) models.DominoPlayWithPass {
	return models.DominoPlayWithPass{
		PlayerPosition: player,
		Bone:           bone,
		Reversed:       bone.Y == edge.X,
	}
}
