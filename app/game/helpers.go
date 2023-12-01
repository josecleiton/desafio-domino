package game

import "github.com/josecleiton/domino/app/models"

func getDuo() models.PlayerPosition {
	return g.Player.Add(2)
}

func handCanPlayThisTurn(
	state *models.DominoGameState,
) ([]models.DominoInTable, []models.DominoInTable) {
	bonesGlueLeft := make([]models.DominoInTable, 0, len(g.Hand))
	bonesGlueRight := make([]models.DominoInTable, 0, len(g.Hand))

	for _, bh := range g.Hand {
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

	g.UnavailableBonesMutex.Lock()
	defer g.UnavailableBonesMutex.Unlock()

	for _, bone := range left {
		if v, ok := g.UnavailableBones[duo][bone.GlueableSide()]; ok && v {
			continue
		}

		duoLeft = append(duoLeft, bone)
	}

	for _, bone := range right {
		if v, ok := g.UnavailableBones[duo][bone.GlueableSide()]; ok && v {
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
	if v, ok := g.UnavailableBones[duo][edge.GlueableSide()]; ok && v {
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
	firstPlayer := g.Player
	passes := 0

	for i := 0; i < models.DominoMaxPlayer; i++ {
		currentPlayer := firstPlayer.Add(i)
		if currentPlayer == getDuo() || currentPlayer == g.Player {
			continue
		}

		ub := g.UnavailableBones[currentPlayer]
		if v, ok := ub[bone.GlueableSide()]; ok && v {
			passes++
		}
	}

	return passes
}

func hasLastBoneEdge(state *models.DominoGameState) models.Edge {

	leftEdge := countBones(state, dominoInTableFromEdge(state, models.LeftEdge))
	rightEdge := countBones(state, dominoInTableFromEdge(state, models.RightEdge))

	if leftEdge == 6 {
		return models.LeftEdge
	}
	if rightEdge == 6 {
		return models.RightEdge
	}
	return models.NoEdge
}
