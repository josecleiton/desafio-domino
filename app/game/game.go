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
var player models.PlayerPosition
var unavailableBones models.UnavailableBonesPlayer

var unavailableBonesMutex sync.Mutex
var intermediateStateWg sync.WaitGroup

func init() {
	states = utils.NewLinkedList[models.DominoGameState]()
	unavailableBones = make(models.UnavailableBonesPlayer, models.DominoMaxPlayer)
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
			currentPlayer := firstPlayer.Add(i)

			if i < nonEvaluatedPlaysLen && nonEvaluatedPlays[i].PlayerPosition == currentPlayer {
				continue
			}

			playerPassed := currentPlayer

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

		allPlays := make([]models.DominoPlay, 0, len(state.Plays))
		allPlays = append(allPlays, state.Plays...)

		defer generateTree(state, guessTreeGenerate{
			Player: player,
			Hand:   hand,
			Plays:  allPlays,
		})

		return models.DominoPlayWithPass{PlayerPosition: player}
	}

	canPlayBoth := leftLen > 0 && rightLen > 0
	if !canPlayBoth {
		play := oneSidedPlay(left, right)
		defer generateTreeByPlay(state, &play)
		return play
	}

	countResult := countPlay(state, left, right)
	if countResult != nil {
		return *countResult
	}

	var duoResult, passedResult *models.DominoPlayWithPass

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()

		intermediateStateWg.Wait()
		duoResult = duoPlay(state, left, right)
	}()

	go func() {
		defer wg.Done()

		intermediateStateWg.Wait()

		passedResult = passedPlay(state, left, right)
	}()

	wg.Wait()

	if duoResult != nil && passedResult != nil {
		passes := countPasses(*passedResult.Bone)

		otherEdge := new(models.DominoInTable)
		if passedResult.Bone.Edge == models.LeftEdge {
			*otherEdge = dominoInTableFromEdge(state, models.RightEdge)
		} else {
			*otherEdge = dominoInTableFromEdge(state, models.LeftEdge)
		}

		duoCanPlayOtherEdge := duoCanPlayEdge(state, *otherEdge)
		if passes > 1 || duoCanPlayOtherEdge {
			return *passedResult
		}

		return *duoResult
	}

	possiblePlays := []*models.DominoPlayWithPass{passedResult, duoResult}

	for _, p := range possiblePlays {
		if p != nil {
			return *p
		}
	}

	log.Println("Something went wrong, no play found")
	return models.DominoPlayWithPass{PlayerPosition: player}

}

func duoCanPlayEdges(state *models.DominoGameState) (bool, bool) {
	unavailableBonesMutex.Lock()
	defer unavailableBonesMutex.Unlock()

	leftEdge, rightEdge := dominoInTableFromEdge(state, models.LeftEdge), dominoInTableFromEdge(state, models.RightEdge)
	return duoCanPlayEdge(state, leftEdge),
		duoCanPlayEdge(state, rightEdge)
}
