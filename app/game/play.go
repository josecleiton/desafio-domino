package game

import (
	"crypto/rand"
	"log"
	"math/big"
	mrand "math/rand"
	"sort"
	"sync"

	"github.com/josecleiton/domino/app/models"
)

var hand []models.Domino
var plays []models.DominoPlayWithPass
var states []models.DominoGameState
var player int
var unavailableBones models.Table

var unavailableBonesMutex sync.Mutex

func Play(state *models.DominoGameState) *models.DominoPlayWithPass {
	states = append(states, *state)
	hand = append(hand, state.Hand...)
	sort.Slice(hand, func(i, j int) bool {
		return hand[i].Sum() >= hand[j].Sum()
	})

	if len(state.Plays) > 0 {
		plays = make([]models.DominoPlayWithPass, 0, len(state.Plays))

		for _, edge := range state.Plays {
			edgePlays := edge.Slice()
			for _, play := range edgePlays {
				plays = append(plays, models.DominoPlayWithPass{
					PlayerPosition: player,
					Bone:           &play.Bone,
				})
			}

		}

		if len(plays) == 0 {
			initialize(state)
		}

		plays = append(plays, midgamePlay(state))
	} else {
		plays = append(plays, initialPlay(state))
	}

	return &plays[len(plays)-1]
}

func initialize(state *models.DominoGameState) {
	player = state.PlayerPosition
	clear(plays)
	clear(states)
}

func initialPlay(state *models.DominoGameState) models.DominoPlayWithPass {
	initialize(state)

	edge := models.LeftEdge
	if cryptoRandSecure(10000)&1 == 1 {
		edge = models.RightEdge
	}

	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone: &models.DominoInTable{
			Edge: edge,
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

func midgamePlay(state *models.DominoGameState) models.DominoPlayWithPass {
	left, right := handCanPlayThisTurn(state)
	leftLen, rightLen := len(left), len(right)

	// pass
	if leftLen == 0 && rightLen == 0 {
		if v, ok := state.Plays[models.LeftEdge]; ok && v != nil {
			unavailableBones[player][v.Tail().Data.Bone.Side()] = true
		}

		if v, ok := state.Plays[models.RightEdge]; ok && v != nil {
			unavailableBones[player][v.Tail().Data.Bone.Side()] = true
		}

		return models.DominoPlayWithPass{PlayerPosition: player}
	}

	canPlayBoth := leftLen > 0 && rightLen > 0
	if !canPlayBoth {
		return oneSidedPlay(left, right)
	}

	var duoPlay, passedPlay, countPlay *models.DominoPlayWithPass

	wg := sync.WaitGroup{}
	wg.Add(3)

	go func() {
		defer wg.Done()

		filteredLeft, filteredRight := duoCanPlayWithBoneGlue(left, right)
		cantPlayLeft, cantPlayRight := len(filteredLeft) == 0, len(filteredRight) == 0
		playsRespectingDuo := make([]models.DominoPlayWithPass, 0, 2)

		// duo cant play with bone glue
		if !cantPlayLeft && !cantPlayRight {
			leftEdge, rightEdge := duoCanPlayEdges(state)

			if leftEdge {
				playsRespectingDuo = append(playsRespectingDuo, playFromDominoInTable(right[0]))
			}

			if rightEdge {
				playsRespectingDuo = append(playsRespectingDuo, playFromDominoInTable(left[0]))
			}

		}

		if cantPlayLeft {
			playsRespectingDuo = append(playsRespectingDuo, playFromDominoInTable(filteredLeft[0]))
		}

		if cantPlayRight {
			playsRespectingDuo = append(playsRespectingDuo, playFromDominoInTable(filteredRight[0]))
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

		if maxBonesLen != 1 {
			sortByPassed(maxBones)
		}

		maxBone := &maxBones[0]

		play := playFromDominoInTable(*maxBone)
		passedPlay = &play
	}()

	go func() {
		defer wg.Done()

		if v, ok := state.Plays[models.LeftEdge]; ok && v != nil {
			leftBonesInGame := countBones(v.Tail().Data.Bone, state)
			if len(left) == 1 && leftBonesInGame == models.DominoUniqueBones-1 {
				play := playFromDominoInTable(right[0])
				countPlay = &play
				return
			}
		}

		if v, ok := state.Plays[models.RightEdge]; ok && v != nil {
			rightBonesInGame := countBones(v.Tail().Data.Bone, state)
			if len(right) == 1 && rightBonesInGame == models.DominoUniqueBones-1 {
				play := playFromDominoInTable(left[0])
				countPlay = &play
			}
		}
	}()

	wg.Wait()

	if countPlay != nil {
		return *countPlay
	}

	if duoPlay != nil && passedPlay != nil {
		passes := countPasses(*passedPlay.Bone)

		var otherEdge models.Edge
		if passedPlay.Bone.Edge == models.LeftEdge {
			otherEdge = models.RightEdge
		} else {
			otherEdge = models.LeftEdge
		}

		duoCanPlayOtherEdge := duoCanPlayEdge(state, otherEdge)
		if passes > 1 || duoCanPlayOtherEdge {
			return *passedPlay
		}

		return *duoPlay
	}

	possiblePlays := []*models.DominoPlayWithPass{passedPlay, duoPlay}

	for _, p := range possiblePlays {
		if p != nil {
			return *p
		}
	}

	log.Println("Something went wrong, no play found")
	return models.DominoPlayWithPass{PlayerPosition: player}

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

func maximizedPlays(playsRespectingDuo []models.DominoPlayWithPass) models.DominoPlayWithPass {
	max := playsRespectingDuo[0]
	for _, play := range playsRespectingDuo[1:] {
		if play.Bone.Sum() > max.Bone.Sum() {
			max = play
		}
	}

	return max
}

func duoCanPlayEdges(state *models.DominoGameState) (bool, bool) {
	unavailableBonesMutex.Lock()
	defer unavailableBonesMutex.Unlock()

	return duoCanPlayEdge(state, models.LeftEdge),
		duoCanPlayEdge(state, models.RightEdge)
}

func duoCanPlayEdge(state *models.DominoGameState, edge models.Edge) bool {
	duo := getDuo()
	if v, ok := state.Plays[edge]; ok && v != nil {
		if v, ok := unavailableBones[duo][v.Tail().Data.Bone.Side()]; ok && v {
			return false
		}
	}

	return true
}

func duoCanPlayWithBoneGlue(left, right []models.DominoInTable) ([]models.DominoInTable, []models.DominoInTable) {
	duo := getDuo()

	duoLeft := make([]models.DominoInTable, 0, len(left))
	duoRight := make([]models.DominoInTable, 0, len(right))

	unavailableBonesMutex.Lock()
	defer unavailableBonesMutex.Unlock()
	for _, bone := range left {
		if v, ok := unavailableBones[duo][bone.Side()]; !(ok && v) {
			duoLeft = append(duoLeft, bone)
		}
	}

	for _, bone := range right {
		if v, ok := unavailableBones[duo][bone.Side()]; !(ok && v) {
			duoRight = append(duoRight, bone)
		}
	}

	return duoLeft, duoRight
}

func handCanPlayThisTurn(state *models.DominoGameState) ([]models.DominoInTable, []models.DominoInTable) {
	bonesGlueLeft := make([]models.DominoInTable, 0, len(hand))
	bonesGlueRight := make([]models.DominoInTable, 0, len(hand))

	for _, h := range hand {
		if v, ok := state.Plays[models.LeftEdge]; ok && v != nil {
			if bone := h.Glue(v.Tail().Data.Bone.Domino); bone != nil {
				bonesGlueLeft = append(bonesGlueLeft, models.DominoInTable{
					Domino: *bone,
					Edge:   models.LeftEdge,
				})
			}
		}

		if v, ok := state.Plays[models.RightEdge]; ok && v != nil {
			if bone := h.Glue(v.Tail().Data.Bone.Domino); bone != nil {
				bonesGlueRight = append(bonesGlueRight, models.DominoInTable{
					Domino: *bone,
					Edge:   models.RightEdge,
				})
			}
		}
	}

	return bonesGlueLeft, bonesGlueRight
}

func oneSidedPlay(left, right []models.DominoInTable) models.DominoPlayWithPass {
	if len(left) > 0 {
		return maximizedPlayWithBones(left)
	}

	return maximizedPlayWithBones(right)
}

func maximizedPlayWithBones(bones []models.DominoInTable) models.DominoPlayWithPass {

	maxBone := bones[0]

	return playFromDominoInTable(maxBone)
}

func playFromDominoInTable(bone models.DominoInTable) models.DominoPlayWithPass {
	return models.DominoPlayWithPass{
		PlayerPosition: player,
		Bone:           &bone,
	}
}

func getDuo() int {
	return ((player + 1) % models.DominoMaxPlayer) + 1
}

func countBones(bone models.DominoInTable, state *models.DominoGameState) int {
	if v, ok := state.Table[bone.Side()]; ok {
		return len(v)
	}

	return 0
}
