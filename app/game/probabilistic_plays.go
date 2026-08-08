package game

import (
	"math/rand"
	"time"

	"github.com/josecleiton/domino/app/models"
)

const monteCarloSimulations = 50

func init() {
	rand.Seed(time.Now().UnixNano())
}

// maximizeWinningChancesPlay evaluates the given plays using Perfect Information Monte Carlo.
func maximizeWinningChancesPlay(
	state *models.DominoGameState,
	plays ...*models.DominoPlayWithPass,
) *models.DominoPlayWithPass {
	if len(plays) == 0 {
		return nil
	}

	delta := models.DominoLength - len(state.Plays) +
		len(state.Hand) + len(g.UnavailableBones[state.PlayerPosition.Next()])
	if delta > 20 {
		// Too early for monte carlo, fallback to a heuristic
		return nil
	}

	playWins := make(map[*models.DominoPlayWithPass]int)
	playDraws := make(map[*models.DominoPlayWithPass]int)

	for i := 0; i < monteCarloSimulations; i++ {
		hands := sampleHands(state)
		if hands == nil {
			continue // failed to sample valid hands
		}

		for _, play := range plays {
			winner, isDraw := simulateDeterminization(state, play, hands)
			if isDraw {
				playDraws[play]++
			} else if winner == g.Player || winner == getDuo() {
				playWins[play]++
			}
		}
	}

	return betterPlayFromPlayDepth(plays, playWins, playDraws)
}

func betterPlayFromPlayDepth(
	plays []*models.DominoPlayWithPass,
	playWins map[*models.DominoPlayWithPass]int,
	playDraws map[*models.DominoPlayWithPass]int,
) *models.DominoPlayWithPass {
	bestPlay := plays[0]
	for _, play := range plays[1:] {
		if playWins[play] > playWins[bestPlay] {
			bestPlay = play
		} else if playWins[play] == playWins[bestPlay] {
			if playDraws[play] > playDraws[bestPlay] {
				bestPlay = play
			}
		}
	}
	if playWins[bestPlay] == 0 && playDraws[bestPlay] == 0 {
		return nil // No wins or draws, let fallback handle it
	}
	return bestPlay
}

func sampleHands(state *models.DominoGameState) map[models.PlayerPosition][]models.Domino {
	g.UnavailableBonesMutex.Lock()
	ub := g.UnavailableBones.Copy()
	g.UnavailableBonesMutex.Unlock()

	cannotPlayMap := make(models.TableMap, models.DominoUniqueBones)
	for i := models.DominoMinBone; i <= models.DominoMaxBone; i++ {
		cannotPlayMap[i] = make(models.TableBone, models.DominoUniqueBones)
	}

	for _, bone := range state.Table {
		cannotPlayMap[bone.L][bone.R] = true
		cannotPlayMap[bone.R][bone.L] = true
	}
	for _, bone := range state.Hand {
		cannotPlayMap[bone.L][bone.R] = true
		cannotPlayMap[bone.R][bone.L] = true
	}

	remainingBones := []models.Domino{}
	for i := models.DominoMinBone; i <= models.DominoMaxBone; i++ {
		for j := i; j <= models.DominoMaxBone; j++ {
			if !cannotPlayMap[i][j] && !cannotPlayMap[j][i] {
				remainingBones = append(remainingBones, models.Domino{L: i, R: j})
			}
		}
	}

	rand.Shuffle(len(remainingBones), func(i, j int) {
		remainingBones[i], remainingBones[j] = remainingBones[j], remainingBones[i]
	})

	hands := make(map[models.PlayerPosition][]models.Domino)
	boneIdx := 0

	for i := 1; i <= models.DominoMaxPlayer; i++ {
		p := models.PlayerPosition(i)
		if p == g.Player {
			continue
		}

		playsCount := 0
		for _, pl := range state.Plays {
			if pl.PlayerPosition == p {
				playsCount++
			}
		}
		bonesNeeded := models.DominoHandLength - playsCount
		if bonesNeeded <= 0 {
			hands[p] = []models.Domino{}
			continue
		}

		playerHand := []models.Domino{}
		for j := 0; j < bonesNeeded; j++ {
			// Find a bone that this player is allowed to have
			found := false
			for k := boneIdx; k < len(remainingBones); k++ {
				bone := remainingBones[k]
				if !ub[p][bone.L] && !ub[p][bone.R] {
					playerHand = append(playerHand, bone)
					remainingBones[boneIdx], remainingBones[k] = remainingBones[k], remainingBones[boneIdx]
					boneIdx++
					found = true
					break
				}
			}
			if !found {
				return nil // Could not satisfy constraints
			}
		}
		hands[p] = playerHand
	}

	return hands
}

// simulateDeterminization returns winner (0 if draw), isDraw
func simulateDeterminization(
	initialState *models.DominoGameState,
	initialPlay *models.DominoPlayWithPass,
	hands map[models.PlayerPosition][]models.Domino,
) (models.PlayerPosition, bool) {
	// 1. Apply initial play
	table := make([]models.Domino, len(initialState.Table))
	copy(table, initialState.Table)

	if initialPlay.Bone != nil {
		if initialPlay.Bone.Edge == models.LeftEdge {
			table = append([]models.Domino{initialPlay.Bone.Domino}, table...)
		} else {
			table = append(table, initialPlay.Bone.Domino)
		}
	}

	myHand := make([]models.Domino, 0, len(initialState.Hand))
	if initialPlay.Bone != nil {
		for _, b := range initialState.Hand {
			if !b.Equals(initialPlay.Bone.Domino) {
				myHand = append(myHand, b)
			}
		}
	} else {
		myHand = append(myHand, initialState.Hand...)
	}
	
	if len(myHand) == 0 {
		return initialState.PlayerPosition, false
	}

	simHands := make(map[models.PlayerPosition][]models.Domino)
	for p, h := range hands {
		ch := make([]models.Domino, len(h))
		copy(ch, h)
		simHands[p] = ch
	}
	simHands[g.Player] = myHand

	currentPlayer := initialState.PlayerPosition.Next()
	passes := 0

	for {
		if passes >= models.DominoMaxPlayer {
			// Draw
			return evaluateDraw(simHands, currentPlayer)
		}

		hand := simHands[currentPlayer]
		var bestPlay *models.DominoInTable

		if len(hand) > 0 {
			leftBone := table[0]
			rightBone := table[len(table)-1]

			for _, bone := range hand {
				leftGlue := models.DominoInTable{Edge: models.LeftEdge, Domino: leftBone}.Glue(bone)
				if leftGlue != nil {
					bestPlay = &models.DominoInTable{Edge: models.LeftEdge, Domino: *leftGlue}
					break
				}
				rightGlue := models.DominoInTable{Edge: models.RightEdge, Domino: rightBone}.Glue(bone)
				if rightGlue != nil {
					bestPlay = &models.DominoInTable{Edge: models.RightEdge, Domino: *rightGlue}
					break
				}
			}
		}

		if bestPlay != nil {
			passes = 0
			if bestPlay.Edge == models.LeftEdge {
				table = append([]models.Domino{bestPlay.Domino}, table...)
			} else {
				table = append(table, bestPlay.Domino)
			}

			newHand := make([]models.Domino, 0, len(hand)-1)
			for _, b := range hand {
				if !b.Equals(bestPlay.Domino) {
					newHand = append(newHand, b)
				}
			}
			simHands[currentPlayer] = newHand

			if len(newHand) == 0 {
				return currentPlayer, false
			}
		} else {
			passes++
		}

		currentPlayer = currentPlayer.Next()
	}
}

func evaluateDraw(hands map[models.PlayerPosition][]models.Domino, blockedPlayer models.PlayerPosition) (models.PlayerPosition, bool) {
	sums := make(map[models.PlayerPosition]int)
	for p, h := range hands {
		for _, b := range h {
			sums[p] += b.Sum()
		}
	}
	duo := getDuo()
	t1 := sums[g.Player] + sums[duo]
	t2 := sums[g.Player.Next()] + sums[duo.Next()]

	if t1 < t2 {
		return g.Player, true
	} else if t2 < t1 {
		return g.Player.Next(), true
	} else {
		// Tie-breaker based on who blocked
		if blockedPlayer != g.Player && blockedPlayer != duo {
			return g.Player, true
		}
		return g.Player.Next(), true
	}
}

// Dummy generateTree functions to satisfy game.go imports before we delete them
func generateTree(state *models.DominoGameState, generate interface{}) {}
func generateTreeByPlay(state *models.DominoGameState, play *models.DominoPlayWithPass) {}
