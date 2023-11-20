package game

import "github.com/josecleiton/domino/app/models"

func getDuo() models.PlayerPosition {
	return player.Add(2)
}

func handCanPlayThisTurn(
	state *models.DominoGameState,
) ([]models.DominoInTable, []models.DominoInTable) {
	bonesGlueLeft := make([]models.DominoInTable, 0, len(playerHand))
	bonesGlueRight := make([]models.DominoInTable, 0, len(playerHand))

	for _, bh := range playerHand {
		left, right := dominoInTableFromEdge(state, models.LeftEdge),
			dominoInTableFromEdge(state, models.RightEdge)
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

func duoCanPlayWithBoneGlue(
	left, right []models.DominoInTable,
) ([]models.DominoInTable, []models.DominoInTable) {
	duo := getDuo()

	duoLeft := make([]models.DominoInTable, 0, len(left))
	duoRight := make([]models.DominoInTable, 0, len(right))

	unavailableBonesMutex.Lock()
	defer unavailableBonesMutex.Unlock()

	for _, bone := range left {
		if v, ok := unavailableBones[duo][bone.GlueableSide()]; ok && v {
			continue
		}

		duoLeft = append(duoLeft, bone)
	}

	for _, bone := range right {
		if v, ok := unavailableBones[duo][bone.GlueableSide()]; ok && v {
			continue
		}

		duoRight = append(duoRight, bone)
	}

	return duoLeft, duoRight
}

func duoCanPlayEdge(
	state *models.DominoGameState,
	edge models.DominoInTable,
) bool {
	duo := getDuo()
	if v, ok := unavailableBones[duo][edge.GlueableSide()]; ok && v {
		return false
	}

	return true
}

func countBones(state *models.DominoGameState, bone models.DominoInTable) int {
	if v, ok := state.TableMap[bone.GlueableSide()]; ok {
		return len(v)
	}

	return 0
}

func countPasses(bone models.DominoInTable) int {
	firstPlayer := player
	passes := 0

	for i := 0; i < models.DominoMaxPlayer; i++ {
		currentPlayer := firstPlayer.Add(i)
		if currentPlayer == getDuo() || currentPlayer == player {
			continue
		}

		ub := unavailableBones[currentPlayer]
		if v, ok := ub[bone.GlueableSide()]; ok && v {
			passes++
		}
	}

	return passes
}
