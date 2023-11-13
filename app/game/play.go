package game

import (
	"crypto/rand"
	"errors"
	"log"
	"math/big"
	mrand "math/rand"
	"sort"
	"sync"

	"github.com/josecleiton/domino/app/models"
)

type nodeDomino struct {
	Left  models.DominoInTable
	Right models.DominoInTable
}

type edgeWithPossibleBones struct {
	Edge  *models.DominoInTable
	Bones []models.DominoInTable
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

	var duoPlay, countPlay *models.DominoPlayWithPass

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		leftEdgeWithPossibleBones, rightEdgeWithPossibleBones := edgeWithPossibleBones{
			Edge:  &node.Left,
			Bones: left,
		}, edgeWithPossibleBones{
			Edge:  &node.Right,
			Bones: right,
		}
		filteredLeft, filteredRight := duoCanPlayWithBoneGlue(leftEdgeWithPossibleBones, rightEdgeWithPossibleBones)
		cantPlayLeft, cantPlayRight := len(filteredLeft) == 0, len(filteredRight) == 0
		playsRespectingDuo := make([]models.DominoPlayWithPass, 0, 2)

		// duo cant play with bone glue
		if !cantPlayLeft && !cantPlayRight {
			leftEdge, rightEdge := duoCanPlayEdge(leftEdgeWithPossibleBones, rightEdgeWithPossibleBones)

			if leftEdge != nil {
				playsRespectingDuo = append(playsRespectingDuo, playFromEdge(right[0], node.Right))
			}

			if rightEdge != nil {
				playsRespectingDuo = append(playsRespectingDuo, playFromEdge(left[0], node.Left))
			}

		}

		if cantPlayLeft {
			playsRespectingDuo = append(playsRespectingDuo, playFromEdge(filteredLeft[0], node.Left))
		}

		if cantPlayRight {
			playsRespectingDuo = append(playsRespectingDuo, playFromEdge(filteredRight[0], node.Right))
		}

		if len(playsRespectingDuo) == 0 {
			return
		}

		maximized := maximizedPlays(playsRespectingDuo)

		duoPlay = &maximized
	}()

	go func() {
		defer wg.Done()

		leftCount, rightCount := append([]models.DominoInTable{}, left...),
			append([]models.DominoInTable{}, right...)

		sortByPassed(leftCount)
		sortByPassed(rightCount)

		maxBones := make([]models.DominoInTable, 0, 2)

		if len(leftCount) > 0 {
			maxBones = append(maxBones, leftCount[0])
		}

		if len(rightCount) > 0 {
			maxBones = append(maxBones, rightCount[0])
		}

		if len(maxBones) == 0 {
			return
		}

		sortByPassed(maxBones)

		countPlay = &models.DominoPlayWithPass{
			PlayerPosition: player,
			Bone:           &maxBones[0],
		}
	}()

	wg.Wait()

	if duoPlay != nil && countPlay != nil {
		// TODO: fazer essa logica de escolhe
		return *duoPlay
	} else if duoPlay != nil {
		return *duoPlay
	}

	return *countPlay
}

func sortByPassed(bones []models.DominoInTable) {
	sort.Slice(bones, func(i, j int) bool {
		return countPasses(bones[i]) >= countPasses(bones[j])
	})
}
func countPasses(bone models.DominoInTable) int {
	currentPlayerIdx := player
	playerIdx := player - 1
	passes := 0

	for i := currentPlayerIdx - 1; i != playerIdx; i = (i + 1) % currentPlayerIdx {
		currentPlayer := i + 1
		if currentPlayer == getDuo() {
			continue
		}

		if v, ok := unavailableBones[currentPlayer][bone.X]; ok && v {
			passes++
		}
	}

	return passes
}

func maximizedPlays(playsRespectingDuo []models.DominoPlayWithPass) models.DominoPlayWithPass {
	max := playsRespectingDuo[0]
	for _, play := range playsRespectingDuo[1:] {
		if play.Bone.Sum() > max.Bone.Sum() {
			max = play
		}
	}

	return max
}

func duoCanPlayEdge(leftEdgeWithPossibleBones, rightEdgeWithPossibleBones edgeWithPossibleBones) (*models.DominoInTable, *models.DominoInTable) {
	leftEdge, rightEdge := leftEdgeWithPossibleBones.Edge, rightEdgeWithPossibleBones.Edge
	duo := getDuo()

	if v, ok := unavailableBones[duo][leftEdge.Side()]; ok && v {
		leftEdge = nil
	}

	if v, ok := unavailableBones[duo][rightEdge.Side()]; ok && v {
		rightEdge = nil
	}

	return leftEdge, rightEdge
}

func duoCanPlayWithBoneGlue(left, right edgeWithPossibleBones) ([]models.DominoInTable, []models.DominoInTable) {
	duo := getDuo()

	duoLeft := make([]models.DominoInTable, 0, len(left.Bones))
	duoRight := make([]models.DominoInTable, 0, len(right.Bones))

	for _, bone := range left.Bones {
		if v, ok := unavailableBones[duo][bone.Side()]; !(ok && v) {
			duoLeft = append(duoLeft, bone)
		}
	}

	for _, bone := range right.Bones {
		if v, ok := unavailableBones[duo][bone.Side()]; !(ok && v) {
			duoRight = append(duoRight, bone)
		}
	}

	return duoLeft, duoRight
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

func handCanPlayThisTurn(edges *nodeDomino) ([]models.DominoInTable, []models.DominoInTable) {
	if edges == nil {
		return nil, nil
	}

	bonesGlueLeft := make([]models.DominoInTable, 0, len(hand))
	bonesGlueRight := make([]models.DominoInTable, 0, len(hand))
	for _, h := range hand {

		if n := node.Left.Glue(h); n != nil {
			bonesGlueLeft = append(bonesGlueLeft, *n)
		}

		if n := node.Right.Glue(h); n != nil {
			bonesGlueRight = append(bonesGlueRight, *n)
		}
	}

	return bonesGlueLeft, bonesGlueRight
}

func oneSidedPlay(left, right []models.DominoInTable) models.DominoPlayWithPass {
	if len(left) > 0 {
		return maximizedPlayWithBones(left, node.Left)
	}

	return maximizedPlayWithBones(right, node.Right)
}

func maximizedPlayWithBones(bones []models.DominoInTable, edge models.DominoInTable) models.DominoPlayWithPass {

	maxBone := bones[0]

	return playFromEdge(maxBone, edge)
}

func playFromEdge(bone, edge models.DominoInTable) models.DominoPlayWithPass {
	return models.DominoPlayWithPass{
		PlayerPosition: player,
		Bone:           &bone,
	}
}

func getDuo() int {
	return ((player + 1) % models.DominoPlayerLength) + 1
}
