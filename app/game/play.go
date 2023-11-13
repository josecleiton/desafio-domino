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

type playPassWithEdge struct {
	Edge *models.DominoInTable
	models.DominoPlayWithPass
}

func (t *nodeDomino) Dominoes() []models.Domino {
	return []models.Domino{t.Left.Domino, t.Right.Domino}
}

var hand []models.Domino
var plays []models.DominoPlayWithPass
var states []models.DominoGameState
var unavailableBones map[int]map[int]bool
var unavailableBonesMutex sync.Mutex
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

	var duoPlay, passedPlay, countPlay *playPassWithEdge

	wg := sync.WaitGroup{}
	wg.Add(3)

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
		playsRespectingDuo := make([]playPassWithEdge, 0, 2)

		// duo cant play with bone glue
		if !cantPlayLeft && !cantPlayRight {
			leftEdge, rightEdge := duoCanPlayEdges(leftEdgeWithPossibleBones, rightEdgeWithPossibleBones)

			if leftEdge != nil {
				playsRespectingDuo = append(playsRespectingDuo, playFromEdge(right[0], &node.Right))
			}

			if rightEdge != nil {
				playsRespectingDuo = append(playsRespectingDuo, playFromEdge(left[0], &node.Left))
			}

		}

		if cantPlayLeft {
			playsRespectingDuo = append(playsRespectingDuo, playFromEdge(filteredLeft[0], &node.Left))
		}

		if cantPlayRight {
			playsRespectingDuo = append(playsRespectingDuo, playFromEdge(filteredRight[0], &node.Right))
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
		leftCountLen, rightCountLen := len(leftCount), len(rightCount)

		if leftCountLen > 0 {
			maxBones = append(maxBones, leftCount[0])
		}

		if rightCountLen > 0 {
			maxBones = append(maxBones, rightCount[0])
		}

		maxBonesLen := len(maxBones)
		if maxBonesLen < 1 {
			return
		}

		var edge *models.DominoInTable
		if maxBonesLen != 1 {
			sortByPassed(maxBones)
		}

		maxBone := &maxBones[0]

		if leftCountLen > 0 && *maxBone == leftCount[0] {
			edge = &node.Left
		} else {
			edge = &node.Right
		}

		passedPlay = &playPassWithEdge{
			DominoPlayWithPass: models.DominoPlayWithPass{
				PlayerPosition: player,
				Bone:           maxBone,
			},
			Edge: edge,
		}
	}()

	go func() {
		defer wg.Done()

		leftBonesInGame := countBones(node.Left, state)
		if len(left) == 1 && leftBonesInGame == models.DominoUniqueBones-1 {
			play := playFromEdge(right[0], &node.Right)
			countPlay = &play
		}

		rightBonesInGame := countBones(node.Right, state)
		if len(right) == 1 && rightBonesInGame == models.DominoUniqueBones-1 {
			play := playFromEdge(left[0], &node.Left)
			countPlay = &play
			return
		}
	}()

	wg.Wait()

	if countPlay != nil {
		return countPlay.DominoPlayWithPass
	}

	if duoPlay != nil && passedPlay != nil {
		passes := countPasses(*passedPlay.Bone)
		var otherEdge *models.DominoInTable

		if passedPlay.Edge == &node.Left {
			otherEdge = &node.Right
		} else {
			otherEdge = &node.Left
		}

		duoCanPlayOtherEdge := duoCanPlayEdge(otherEdge)
		if passes > 1 || duoCanPlayOtherEdge {
			return passedPlay.DominoPlayWithPass
		}

		return duoPlay.DominoPlayWithPass
	}

	possiblePlays := []*playPassWithEdge{passedPlay, duoPlay, countPlay}

	for _, p := range possiblePlays {
		if p != nil {
			return p.DominoPlayWithPass
		}
	}

	return models.DominoPlayWithPass{
		PlayerPosition: player,
	}

}

func sortByPassed(bones []models.DominoInTable) {
	unavailableBonesMutex.Lock()
	defer unavailableBonesMutex.Unlock()
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

func maximizedPlays(playsRespectingDuo []playPassWithEdge) playPassWithEdge {
	max := playsRespectingDuo[0]
	for _, play := range playsRespectingDuo[1:] {
		if play.Bone.Sum() > max.Bone.Sum() {
			max = play
		}
	}

	return max
}

func duoCanPlayEdges(leftEdgeWithPossibleBones, rightEdgeWithPossibleBones edgeWithPossibleBones) (*models.DominoInTable, *models.DominoInTable) {
	unavailableBonesMutex.Lock()
	defer unavailableBonesMutex.Unlock()

	leftEdge, rightEdge := leftEdgeWithPossibleBones.Edge, rightEdgeWithPossibleBones.Edge

	if duoCanPlayEdge(leftEdge) {
		leftEdge = nil
	}

	if duoCanPlayEdge(rightEdge) {
		rightEdge = nil
	}

	return leftEdge, rightEdge
}

func duoCanPlayEdge(edge *models.DominoInTable) bool {
	duo := getDuo()
	if v, ok := unavailableBones[duo][edge.Side()]; ok && v {
		return false
	}

	return true
}

func duoCanPlayWithBoneGlue(left, right edgeWithPossibleBones) ([]models.DominoInTable, []models.DominoInTable) {
	duo := getDuo()

	duoLeft := make([]models.DominoInTable, 0, len(left.Bones))
	duoRight := make([]models.DominoInTable, 0, len(right.Bones))

	unavailableBonesMutex.Lock()
	defer unavailableBonesMutex.Unlock()
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
		return maximizedPlayWithBones(left, &node.Left)
	}

	return maximizedPlayWithBones(right, &node.Right)
}

func maximizedPlayWithBones(bones []models.DominoInTable, edge *models.DominoInTable) models.DominoPlayWithPass {

	maxBone := bones[0]

	return playFromEdge(maxBone, edge).DominoPlayWithPass
}

func playFromEdge(bone models.DominoInTable, edge *models.DominoInTable) playPassWithEdge {
	return playPassWithEdge{
		DominoPlayWithPass: models.DominoPlayWithPass{
			PlayerPosition: player,
			Bone:           &bone,
		},
		Edge: edge,
	}
}

func getDuo() int {
	return ((player + 1) % models.DominoPlayerLength) + 1
}

func countBones(bone models.DominoInTable, state *models.DominoGameState) int {
	if v, ok := state.Table[bone.Side()]; ok {
		return len(v)
	}

	return 0
}
