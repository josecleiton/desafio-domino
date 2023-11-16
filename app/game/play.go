package game

import (
	"log"
	"sort"
	"sync"

	"github.com/josecleiton/domino/app/models"
	"github.com/josecleiton/domino/app/utils"
)

var hand []models.Domino
var plays []models.DominoPlayWithPass
var states *utils.LinkedList[models.DominoGameState]
var player int
var unavailableBones models.TableMap

var unavailableBonesMutex sync.Mutex
var intermediateStateWg sync.WaitGroup

func init() {
	states = utils.NewLinkedList[models.DominoGameState]()
	unavailableBones = make(models.TableMap, models.DominoMaxPlayer)
	player = -1
}

func Play(state *models.DominoGameState) models.DominoPlayWithPass {
	hasToInitialize := false
	if player != state.PlayerPosition {
		hasToInitialize = true
	}

	if !hasToInitialize {
		if s := states.HeadSafe(); s == nil || len(s.Prev.Data.Plays) > len(state.Plays) {
			hasToInitialize = true
		}
	}

	if hasToInitialize {
		initialize(state)
	}

	states.Push(state)
	forLimit := states.Len() - models.DominoMaxPlayer
	for i := 0; i < forLimit; i++ {
		states.PopFront()
	}
	hand = make([]models.Domino, 0, len(state.Hand))
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

		plays = append(plays, midgamePlay(state))
	} else {
		plays = append(plays, initialPlay(state))
	}

	return plays[len(plays)-1]
}

func intermediateStates(state *models.DominoGameState) {
	current := states.HeadSafe()
	statesLen := states.Len()
	for i := 0; current != nil && i < statesLen; i++ {
		st := current.Data

		var nd *models.DominoGameState
		if current.Next != nil {
			nd = current.Next.Data
		}

		stPlaysLen := len(st.Plays)
		if nd == nil || stPlaysLen == 0 || stPlaysLen == len(nd.Plays) {
			continue
		}

		stPlayIdx := -1
		for i := len(nd.Plays) - 1; i >= 0; i-- {
			if nd.Plays[i] == st.Plays[stPlaysLen-1] {
				stPlayIdx = i
				break
			}

		}

		nonEvaluatedPlays := nd.Plays[stPlayIdx+1:]
		nonEvaluatedPlaysLen := len(nonEvaluatedPlays)
		if nonEvaluatedPlaysLen == models.DominoMaxPlayer {
			continue
		}

		firstPlayer := nonEvaluatedPlays[0].PlayerPosition

		for i := 0; i < models.DominoMaxPlayer; i++ {
			currentPlayerIdx := (firstPlayer + i - 1) % models.DominoMaxPlayer

			if i < nonEvaluatedPlaysLen && nonEvaluatedPlays[i].PlayerPosition-1 == currentPlayerIdx {
				continue
			}

			playerPassed := currentPlayerIdx + 1

			if _, ok := unavailableBones[playerPassed]; !ok {
				unavailableBones[playerPassed] = make(models.TableBone, models.DominoUniqueBones)
			}

			currentPlayOnEdgeIdx := min(nonEvaluatedPlaysLen-1, i) + stPlayIdx + 1
			currentPlayOnEdge := nd.Plays[currentPlayOnEdgeIdx]
			var playOnReversedEdge *models.DominoPlay
			for i := currentPlayOnEdgeIdx - 1; i >= 0; i-- {
				play := &nd.Plays[i]
				if play.Bone.Edge == currentPlayOnEdge.Bone.Edge {
					continue
				}
				playOnReversedEdge = play
				break
			}

			unavailableBones[playerPassed][currentPlayOnEdge.Bone.GlueableSide()] = true
			unavailableBones[playerPassed][playOnReversedEdge.Bone.GlueableSide()] = true
		}

		current = current.Next
	}

	for states.Len() > 1 {
		states.PopFront()
	}
}

func initialize(state *models.DominoGameState) {
	player = state.PlayerPosition
	clear(plays)
	clear(unavailableBones)
	states.Clear()
}

func initialPlay(state *models.DominoGameState) models.DominoPlayWithPass {
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
			unavailableBones[player][ep.Y] = true
		}

		if ep, ok := state.Edges[models.RightEdge]; ok && ep != nil {
			unavailableBones[player][ep.Y] = true
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

		otherEdge := new(models.DominoInTable)
		if passedPlay.Bone.Edge == models.LeftEdge {
			*otherEdge = dominoInTableFromEdge(state, models.RightEdge)
		} else {
			*otherEdge = dominoInTableFromEdge(state, models.LeftEdge)
		}

		duoCanPlayOtherEdge := duoCanPlayEdge(state, *otherEdge)
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
		leftBonesInGame := countBones(state, models.DominoInTable{
			Edge: models.LeftEdge,
			Domino: models.Domino{
				X: ep.Y,
				Y: ep.X,
			},
		})
		if len(left) == 1 && leftBonesInGame == models.DominoUniqueBones-1 {
			play := playFromDominoInTable(right[0])
			return &play
		}
	}

	if ep, ok := state.Edges[models.RightEdge]; ok && ep != nil {
		rightBonesInGame := countBones(state, models.DominoInTable{
			Edge: models.RightEdge,
			Domino: models.Domino{
				X: ep.X,
				Y: ep.Y,
			},
		})
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
	firstPlayerIdx := player - 1
	passes := 0

	for i := 0; i < models.DominoMaxPlayer; i++ {
		currentPlayer := (firstPlayerIdx+i)%models.DominoMaxPlayer + 1
		if currentPlayer == getDuo() || currentPlayer == player {
			continue
		}

		if v, ok := unavailableBones[currentPlayer][bone.GlueableSide()]; ok && v {
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

	leftEdge, rightEdge := dominoInTableFromEdge(state, models.LeftEdge), dominoInTableFromEdge(state, models.RightEdge)
	return duoCanPlayEdge(state, leftEdge),
		duoCanPlayEdge(state, rightEdge)
}

func dominoInTableFromEdge(state *models.DominoGameState, edge models.Edge) models.DominoInTable {
	bone := state.Edges[edge]
	return models.DominoInTable{
		Edge: edge,
		Domino: models.Domino{
			X: bone.X,
			Y: bone.Y,
		},
	}
}

func duoCanPlayEdge(state *models.DominoGameState, edge models.DominoInTable) bool {
	duo := getDuo()
	if v, ok := unavailableBones[duo][edge.GlueableSide()]; ok && v {
		return false
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
		left, right := dominoInTableFromEdge(state, models.LeftEdge), dominoInTableFromEdge(state, models.RightEdge)
		if bone := left.Glue(bh); bone != nil {
			bonesGlueLeft = append(bonesGlueLeft, models.DominoInTable{
				Domino: *bone,
				Edge:   models.LeftEdge,
			})
		}

		if bone := right.Glue(bh); bone != nil {
			bonesGlueRight = append(bonesGlueRight, models.DominoInTable{
				Domino: *bone,
				Edge:   models.RightEdge,
			})
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
	if v, ok := state.TableMap[bone.GlueableSide()]; ok {
		return len(v)
	}

	return 0
}
