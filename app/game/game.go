package game

import (
	"container/ring"
	"log"
	"sort"
	"sync"

	"github.com/josecleiton/domino/app/models"
)

type gameGlobals struct {
	Hand             []models.Domino
	States           *ring.Ring
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
		if g.States.Len() > 0 {
			if prevState := g.States.Prev(); prevState == nil ||
				len(prevState.Value.(*models.DominoGameState).Plays) > len(state.Plays) {
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

		g.States = g.States.Link(otherRing)
	}

	forLimit := g.States.Len() - models.DominoMaxPlayer
	if forLimit > 0 {
		g.States.Prev().Unlink(forLimit)
	}

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
	current := g.States
	statesLen := g.States.Len()
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

			if _, ok := g.UnavailableBones[playerPassed]; !ok {
				g.UnavailableBones[playerPassed] = make(
					models.TableBone,
					models.DominoUniqueBones,
				)
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

			g.UnavailableBones[playerPassed][currentPlayOnEdge.Bone.GlueableSide()] = true
			g.UnavailableBones[playerPassed][playOnReversedEdge.Bone.GlueableSide()] = true
		}

		current = current.Next()
	}

	for g.States.Len() > 1 {
		g.States.Prev().Unlink(1)
	}
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

	g.States = ring.New(1)
	g.States.Value = state
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

	edge := hasLastBoneEdge(state)
	if edge != models.NoEdge {
		if edge == models.LeftEdge {
			play := commonMaximizedPlay(right)
			generateTreeByPlay(state, &play)
			return play
		}
		if edge == models.RightEdge {
			play := commonMaximizedPlay(left)
			generateTreeByPlay(state, &play)
			return play
		}
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
