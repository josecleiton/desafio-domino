package game

import (
	"sort"

	"github.com/josecleiton/domino/app/models"
)

func oneSidedPlay(left, right []models.DominoInTable) models.DominoPlayWithPass {
	if len(left) > 0 {
		return commonMaximizedPlay(left)
	}

	return commonMaximizedPlay(right)
}

func commonMaximizedPlay(bones []models.DominoInTable) models.DominoPlayWithPass {

	commonBones := make([]indexedCount, models.DominoUniqueBones)
	for _, eb := range bones {
		for _, hb := range hand {
			if bone := eb.Glue(hb); bone != nil {
				side := eb.GlueableSide()
				commonBones[side].Count++
				commonBones[side].Idx = side
			}
		}
	}

	sort.Slice(commonBones, func(i, j int) bool {
		return commonBones[i].Count >= commonBones[j].Count
	})

	if commonBones[0].Count != 1 {
		side := commonBones[0].Idx
		for _, bone := range bones {
			if bone.GlueableSide() != side {
				continue
			}

			return playFromDominoInTable(bone)
		}
	}

	return playFromDominoInTable(bones[0])
}

func maximizedPlays(playsRespectingDuo []models.DominoPlayWithPass) *models.DominoPlayWithPass {
	max := playsRespectingDuo[0]
	for _, play := range playsRespectingDuo[1:] {
		if play.Bone.Sum() > max.Bone.Sum() {
			max = play
		}
	}

	return &max
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
		if len(left)+leftBonesInGame == models.DominoUniqueBones {
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
		if len(right)+rightBonesInGame == models.DominoUniqueBones {
			play := playFromDominoInTable(left[0])
			return &play
		}
	}

	return nil
}

func duoPlay(state *models.DominoGameState, left, right []models.DominoInTable) *models.DominoPlayWithPass {

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
		playsRespectingDuo = append(playsRespectingDuo, playFromDominoInTable(filteredRight[0]))
	}

	if cantPlayRight {
		playsRespectingDuo = append(playsRespectingDuo, playFromDominoInTable(filteredLeft[0]))
	}

	if len(playsRespectingDuo) == 0 {
		return nil
	}

	return maximizedPlays(playsRespectingDuo)

}

func passedPlay(state *models.DominoGameState, left, right []models.DominoInTable) *models.DominoPlayWithPass {
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
		return nil
	}

	if maxBonesLen != 1 {
		sortByPassed(maxBones)
	}

	maxBone := &maxBones[0]

	play := playFromDominoInTable(*maxBone)

	return &play
}
