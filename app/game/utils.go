package game

import (
	"sort"

	"github.com/josecleiton/domino/app/models"
)

type indexedCount struct {
	Idx   int
	Count int
}

func playFromDominoInTable(bone models.DominoInTable) models.DominoPlayWithPass {
	return models.DominoPlayWithPass{
		PlayerPosition: player,
		Bone:           &bone,
	}
}

func dominoInTableFromEdge(
	state *models.DominoGameState,
	edge models.Edge,
) models.DominoInTable {
	bone := state.Edges()[edge]
	return models.DominoInTable{
		Edge: edge,
		Domino: models.Domino{
			X: bone.X,
			Y: bone.Y,
		},
	}
}

func dominoInTableFromDomino(
	domino models.Domino,
	edge models.Edge,
) models.DominoInTable {
	return models.DominoInTable{
		Edge: edge,
		Domino: models.Domino{
			X: domino.X,
			Y: domino.Y,
		},
	}
}

func sortByPassed(bones []models.DominoInTable) {
	unavailableBonesMutex.Lock()
	defer unavailableBonesMutex.Unlock()
	sort.Slice(bones, func(i, j int) bool {
		return countPasses(bones[i]) >= countPasses(bones[j])
	})
}

func tableMapFromDominoes(dominoes []models.Domino) models.TableMap {
	table := make(models.TableMap, models.DominoUniqueBones)
	for _, domino := range dominoes {
		if _, ok := table[domino.X]; !ok {
			table[domino.X] = make(models.TableBone, models.DominoUniqueBones)
		}

		if _, ok := table[domino.Y]; !ok {
			table[domino.Y] = make(models.TableBone, models.DominoUniqueBones)
		}

		table[domino.X][domino.Y] = true
		table[domino.Y][domino.X] = true
	}

	return table
}
