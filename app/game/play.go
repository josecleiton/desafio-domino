package game

import (
	"log"
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
var intermediateStateWg sync.WaitGroup

func Play(state *models.DominoGameState) *models.DominoPlayWithPass {
	states = append(states, *state)
	states = states[len(states)-models.DominoMaxPlayer:]

	hand = append(hand, state.Hand...)
	sort.Slice(hand, func(i, j int) bool {
		return hand[i].Sum() >= hand[j].Sum()
	})

	if len(state.Plays) > 0 {
		intermediateStateWg.Add(1)
		go func() {
			defer intermediateStateWg.Done()
			intermediateStates(state)
		}()

		plays = make([]models.DominoPlayWithPass, 0, len(state.Edges))

		for _, play := range state.Plays {
			if play.PlayerPosition != player {
				continue
			}
			plays = append(plays, models.DominoPlayWithPass{
				PlayerPosition: player,
				Bone:           &play.Bone,
			})

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

func intermediateStates(state *models.DominoGameState) {
	if len(states) == 1 {
		return
	}

	currentPlay := state.Plays[len(state.Plays)-1]

	for i := 0; i < len(states); i++ {
		j := i + 1
		st, nd := &states[i], &states[j]

		if nd == nil && len(states) > 2 {
			nd = st
			st = &states[i-1]
		}

		if nd == nil ||
			st == nil || len(st.Plays) == 0 ||
			st.PlayerPosition == player || nd.PlayerPosition == player {
			continue
		}

		lastStPlay, lastNdPlay := st.Plays[len(st.Plays)-1], nd.Plays[len(nd.Plays)-1]

		if lastStPlay == lastNdPlay {
			continue
		}

		currentPlayerIdx := lastStPlay.PlayerPosition - 1
		playerIdx := player - 1
		for i := currentPlayerIdx - 1; i != playerIdx; i = (i + 1) % currentPlayerIdx {
			playerPassed := i + 1

			for _, bone := range st.Edges.Bones() {
				unavailableBones[playerPassed][bone.X] = true
				unavailableBones[playerPassed][bone.Y] = true
			}
		}

		i++
	}

	currentPlayerIdx := currentPlay.PlayerPosition - 1
	playerIdx := player - 1
	for i := currentPlayerIdx - 1; i != playerIdx; i = (i + 1) % currentPlayerIdx {
		playerPassed := i + 1

		for _, bone := range state.Edges.Bones() {
			unavailableBones[playerPassed][bone.X] = true
			unavailableBones[playerPassed][bone.Y] = true
		}
	}

	states = states[len(states)-1:]
}

func initialize(state *models.DominoGameState) {
	player = state.PlayerPosition
	clear(plays)
	clear(states)
}

func initialPlay(state *models.DominoGameState) models.DominoPlayWithPass {
	initialize(state)

	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone: &models.DominoInTable{
			Edge: models.LeftEdge,
			Domino: models.Domino{
				X: hand[0].X,
				Y: hand[0].Y,
			},
		},
	}
}

func midgamePlay(state *models.DominoGameState) models.DominoPlayWithPass {
	left, right := handCanPlayThisTurn(state)
	leftLen, rightLen := len(left), len(right)

	// pass
	if leftLen == 0 && rightLen == 0 {
		if ep, ok := state.Edges[models.LeftEdge]; ok && ep != nil {
			unavailableBones[player][ep.Bone.Y] = true
		}

		if ep, ok := state.Edges[models.RightEdge]; ok && ep != nil {
			unavailableBones[player][ep.Bone.Y] = true
		}

		return models.DominoPlayWithPass{PlayerPosition: player}
	}

	canPlayBoth := leftLen > 0 && rightLen > 0
	if !canPlayBoth {
		return oneSidedPlay(left, right)
	}

	count := countPlay(state, left, right)
	if count != nil {
		return *count
	}

	var duoPlay, passedPlay *models.DominoPlayWithPass

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		intermediateStateWg.Wait()

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

		intermediateStateWg.Wait()

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

	wg.Wait()

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

func countPlay(state *models.DominoGameState, left, right []models.DominoInTable) *models.DominoPlayWithPass {
	if ep, ok := state.Edges[models.LeftEdge]; ok && ep != nil {
		leftBonesInGame := countBones(state, ep.Bone)
		if len(left) == 1 && leftBonesInGame == models.DominoUniqueBones-1 {
			play := playFromDominoInTable(right[0])
			return &play
		}
	}

	if ep, ok := state.Edges[models.RightEdge]; ok && ep != nil {
		rightBonesInGame := countBones(state, ep.Bone)
		if len(right) == 1 && rightBonesInGame == models.DominoUniqueBones-1 {
			play := playFromDominoInTable(left[0])
			return &play
		}
	}

	return nil
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
	if ep, ok := state.Edges[edge]; ok && ep != nil {
		if v, ok := unavailableBones[duo][ep.Bone.Y]; ok && v {
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
		if v, ok := unavailableBones[duo][bone.Y]; ok && v {
			continue
		}

		duoLeft = append(duoLeft, bone)
	}

	for _, bone := range right {
		if v, ok := unavailableBones[duo][bone.Y]; ok && v {
			continue
		}

		duoRight = append(duoRight, bone)
	}

	return duoLeft, duoRight
}

func handCanPlayThisTurn(state *models.DominoGameState) ([]models.DominoInTable, []models.DominoInTable) {
	bonesGlueLeft := make([]models.DominoInTable, 0, len(hand))
	bonesGlueRight := make([]models.DominoInTable, 0, len(hand))

	for _, bh := range hand {
		if ep, ok := state.Edges[models.LeftEdge]; ok && ep != nil {
			if bone := bh.Glue(ep.Bone.Domino); bone != nil {
				bonesGlueLeft = append(bonesGlueLeft, models.DominoInTable{
					Domino: *bone,
					Edge:   models.LeftEdge,
				})
			}
		}

		if ep, ok := state.Edges[models.RightEdge]; ok && ep != nil {
			if bone := bh.Glue(ep.Bone.Domino); bone != nil {
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

func countBones(state *models.DominoGameState, bone models.DominoInTable) int {
	if v, ok := state.Table[bone.Y]; ok {
		return len(v)
	}

	return 0
}
