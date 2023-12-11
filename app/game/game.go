package game

import (
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/josecleiton/domino/app/models"
)

type gameGlobals struct {
	Hand             []models.Domino
	States           []models.DominoGameState
	Player           models.PlayerPosition
	UnavailableBones models.UnavailableBonesPlayer

	// sync
	UnavailableBonesMutex sync.Mutex
	IntermediateStateWg   sync.WaitGroup
}

var g *gameGlobals

func init() {
	g = &gameGlobals{
		Hand: []models.Domino{},
	}

}

func Play(state *models.DominoGameState) models.DominoPlayWithPass {
	hasToInitialize := false
	if g.Player != state.PlayerPosition {
		hasToInitialize = true
	}

	if !hasToInitialize {
		statesLen := len(g.States)
		if statesLen > 0 && len(state.Plays) > len(g.States[statesLen-1].Plays) {
			hasToInitialize = true
		}
	}

	if hasToInitialize {
		initialize(state)
	} else {
		g.States = append(g.States, *state)
	}

	// forLimit := len(g.States) - models.DominoMaxPlayer - 1
	// if forLimit > 0 {
	// 	g.States = g.States[forLimit:]
	// }

	g.Hand = make([]models.Domino, 0, len(state.Hand))
	g.Hand = append(g.Hand, state.Hand...)
	sort.Slice(g.Hand, func(i, j int) bool {
		return g.Hand[i].Sum() >= g.Hand[j].Sum()
	})

	if len(state.Plays) > 0 {
		g.IntermediateStateWg.Add(1)
		go func() {
			defer g.IntermediateStateWg.Done()
			intermediateStates(state)
		}()

		return midgamePlay(state)
	}

	return initialPlay(state)
}

func intermediateStates(state *models.DominoGameState) {
	statesLen := len(g.States)
	last := &g.States[statesLen-1]
	for i := statesLen - 2; i >= 0; i++ {
		first := &g.States[i]

		lastPlaysLen := len(last.Plays)
		if lastPlaysLen == 0 || lastPlaysLen == len(first.Plays) {
			continue
		}

		stPlayIdx := -1
		for i := len(first.Plays) - 1; i >= 0; i-- {
			if first.Plays[i] == last.Plays[lastPlaysLen-1] {
				stPlayIdx = i
				break
			}

		}

		if stPlayIdx == -1 {
			log.Println("Something went wrong, no play found")
			continue
		}

		fmt.Println("ndPlay", last.Plays)
		fmt.Println("stPlay", last.Plays[stPlayIdx])

		nonEvaluatedPlays := last.Plays[stPlayIdx+1:]
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

			if _, ok := g.UnavailableBones[playerPassed]; !ok {
				g.UnavailableBones[playerPassed] = make(
					models.TableBone,
					models.DominoUniqueBones,
				)
			}

			currentPlayOnEdgeIdx := min(nonEvaluatedPlaysLen-1, i) + stPlayIdx + 1
			currentPlayOnEdge := first.Plays[currentPlayOnEdgeIdx]
			var playOnReversedEdge *models.DominoPlay
			for i := currentPlayOnEdgeIdx - 1; i >= 0; i-- {
				play := &first.Plays[i]
				if play.Bone.Edge == currentPlayOnEdge.Bone.Edge {
					continue
				}
				playOnReversedEdge = play
				break
			}

			fmt.Println(currentPlayOnEdge)
			fmt.Println(playOnReversedEdge)
			fmt.Println(g.UnavailableBones[playerPassed])
			g.UnavailableBones[playerPassed][currentPlayOnEdge.Bone.GlueableSide()] = true
			g.UnavailableBones[playerPassed][playOnReversedEdge.Bone.GlueableSide()] = true
		}
	}

	g.States = g.States[statesLen-1:]
}

func initialize(state *models.DominoGameState) {
	g.Player = state.PlayerPosition

	g.UnavailableBones = make(
		models.UnavailableBonesPlayer,
		models.DominoMaxPlayer,
	)
	for i := models.DominoMinPlayer; i <= models.DominoMaxPlayer; i++ {
		player := models.PlayerPosition(i)
		g.UnavailableBones[player] = make(
			models.TableBone,
			models.DominoUniqueBones,
		)
	}

	g.States = make([]models.DominoGameState, 0, models.DominoLength)
	g.States = append(g.States, *state)
}

func initialPlay(state *models.DominoGameState) models.DominoPlayWithPass {
	return models.DominoPlayWithPass{
		PlayerPosition: state.PlayerPosition,
		Bone: &models.DominoInTable{
			Edge: models.LeftEdge,
			Domino: models.Domino{
				L: g.Hand[0].L,
				R: g.Hand[0].R,
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
			g.UnavailableBones[g.Player][ep.R] = true
		}

		if ep, ok := edges[models.RightEdge]; ok && ep != nil {
			g.UnavailableBones[g.Player][ep.R] = true
		}

		allPlays := make([]models.DominoPlay, 0, len(state.Plays))
		allPlays = append(allPlays, state.Plays...)

		generateTree(state, guessTreeGenerate{
			Player: g.Player,
			Hand:   g.Hand,
			Plays:  allPlays,
		})

		return models.DominoPlayWithPass{PlayerPosition: g.Player}
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

		g.IntermediateStateWg.Wait()
		duoResult = duoPlay(state, left, right)
	}()

	go func() {
		defer wg.Done()

		g.IntermediateStateWg.Wait()

		passedResult = passedPlay(state, left, right)
	}()

	wg.Wait()

	if duoResult != nil && passedResult != nil {
		// if maximizedPlay := maximizeWinningChancesPlay(duoResult, passedResult); maximizedPlay != nil {
		// 	return *maximizedPlay
		// }

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
	return models.DominoPlayWithPass{PlayerPosition: g.Player}

}

func duoCanPlayEdges(state *models.DominoGameState) (bool, bool) {
	g.UnavailableBonesMutex.Lock()
	defer g.UnavailableBonesMutex.Unlock()

	leftEdge, rightEdge := dominoInTableFromEdge(state, models.LeftEdge),
		dominoInTableFromEdge(state, models.RightEdge)

	return duoCanPlayEdge(state, leftEdge),
		duoCanPlayEdge(state, rightEdge)
}
