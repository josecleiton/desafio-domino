package game

import (
	"container/ring"
	"log"
	"sort"
	"sync"

	"github.com/josecleiton/domino/app/models"
)

var playerHand []models.Domino
var states *ring.Ring
var player models.PlayerPosition
var unavailableBones models.UnavailableBonesPlayer

var unavailableBonesMutex sync.Mutex
var intermediateStateWg sync.WaitGroup

func init() {
	unavailableBones = make(models.UnavailableBonesPlayer, models.DominoMaxPlayer)
}

func Play(state *models.DominoGameState) models.DominoPlayWithPass {
	hasToInitialize := false
	if player != state.PlayerPosition {
		hasToInitialize = true
	}

	if !hasToInitialize {
		if states.Len() > 0 {
			if prevState := states.Prev(); prevState == nil || len(prevState.Value.(*models.DominoGameState).Plays) > len(state.Plays) {
				hasToInitialize = true
			}
		} else {
			hasToInitialize = true
		}
	}

	if hasToInitialize {
		initialize(state)
	} else {
		otherRing := ring.New(1)
		otherRing.Value = state

		states = states.Link(otherRing)
	}

	forLimit := states.Len() - models.DominoMaxPlayer
	if forLimit > 0 {
		states.Prev().Unlink(forLimit)
	}

	playerHand = make([]models.Domino, 0, len(state.Hand))
	playerHand = append(playerHand, state.Hand...)
	sort.Slice(playerHand, func(i, j int) bool {
		return playerHand[i].Sum() >= playerHand[j].Sum()
	})

	if len(state.Plays) > 0 {
		intermediateStateWg.Add(1)
		go func() {
			defer intermediateStateWg.Done()
			intermediateStates(state)
		}()

		return midgamePlay(state)
	}

	return initialPlay(state)
}

func intermediateStates(state *models.DominoGameState) {
	current := states
	statesLen := states.Len()
	for i := 0; current != nil && i < statesLen; i++ {
		st := current.Value.(*models.DominoGameState)

		var nd *models.DominoGameState
		if current.Next() != nil {
			nd = current.Next().Value.(*models.DominoGameState)
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

		current = current.Next()
	}

	for states.Len() > 1 {
		states.Prev().Unlink(1)
	}
}

func initialize(state *models.DominoGameState) {
	player = state.PlayerPosition
	clear(unavailableBones)
	states = ring.New(1)
	states.Value = state
}

func initialPlay(state *models.DominoGameState) models.DominoPlayWithPass {
	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone: &models.DominoInTable{
			Edge: models.LeftEdge,
			Domino: models.Domino{
				X: playerHand[0].X,
				Y: playerHand[0].Y,
			},
		},
	}
}

func midgamePlay(state *models.DominoGameState) models.DominoPlayWithPass {
	left, right := handCanPlayThisTurn(state)
	leftLen, rightLen := len(left), len(right)

	// pass
	if leftLen == 0 && rightLen == 0 {
		edges := state.Edges()
		if ep, ok := edges[models.LeftEdge]; ok && ep != nil {
			unavailableBones[player][ep.Y] = true
		}

		if ep, ok := edges[models.RightEdge]; ok && ep != nil {
			unavailableBones[player][ep.Y] = true
		}

		allPlays := make([]models.DominoPlay, 0, len(state.Plays))
		allPlays = append(allPlays, state.Plays...)

		generateTree(state, guessTreeGenerate{
			Player: player,
			Hand:   playerHand,
			Plays:  allPlays,
		})

		return models.DominoPlayWithPass{PlayerPosition: player}
	}

	canPlayBoth := leftLen > 0 && rightLen > 0
	if !canPlayBoth {
		play := oneSidedPlay(left, right)
		generateTreeByPlay(state, &play)
		return play
	}

	countResult := countPlay(state, left, right)
	if countResult != nil {
		generateTreeByPlay(state, countResult)
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
		if maximizedPlay := maximizeWinningChancesPlay(duoResult, passedResult); maximizedPlay != nil {
			return *maximizedPlay
		}

		passes := countPasses(*passedResult.Bone)

		otherEdge := new(models.DominoInTable)
		if passedResult.Bone.Edge == models.LeftEdge {
			*otherEdge = dominoInTableFromEdge(state, models.RightEdge)
		} else {
			*otherEdge = dominoInTableFromEdge(state, models.LeftEdge)
		}

		play := duoResult

		duoCanPlayOtherEdge := duoCanPlayEdge(state, *otherEdge)
		if passes > 1 || duoCanPlayOtherEdge {
			play = passedResult
		}

		generateTreeByPlay(state, play)
		treeGeneratingWg.Wait()

		return *play
	}

	possiblePlays := []*models.DominoPlayWithPass{passedResult, duoResult}

	for _, p := range possiblePlays {
		if p == nil {
			continue
		}

		generateTreeByPlay(state, p)
		return *p
	}

	log.Println("Something went wrong, no play found")
	return models.DominoPlayWithPass{PlayerPosition: player}

}

func duoCanPlayEdges(state *models.DominoGameState) (bool, bool) {
	unavailableBonesMutex.Lock()
	defer unavailableBonesMutex.Unlock()

	leftEdge, rightEdge := dominoInTableFromEdge(state, models.LeftEdge),
		dominoInTableFromEdge(state, models.RightEdge)

	return duoCanPlayEdge(state, leftEdge),
		duoCanPlayEdge(state, rightEdge)
}
