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
		PlayerPosition: g.Player,
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
			L: bone.L,
			R: bone.R,
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
			L: domino.L,
			R: domino.R,
		},
	}
}

func sortByPassed(bones []models.DominoInTable) {
	g.UnavailableBonesMutex.Lock()
	defer g.UnavailableBonesMutex.Unlock()
	sort.Slice(bones, func(i, j int) bool {
		return countPasses(bones[i]) >= countPasses(bones[j])
	})
}

func tableMapFromDominoes(dominoes []models.Domino) models.TableMap {
	table := make(models.TableMap, models.DominoUniqueBones)
	for _, domino := range dominoes {
		if _, ok := table[domino.L]; !ok {
			table[domino.L] = make(models.TableBone, models.DominoUniqueBones)
		}

		if _, ok := table[domino.R]; !ok {
			table[domino.R] = make(models.TableBone, models.DominoUniqueBones)
		}

		table[domino.L][domino.R] = true
		table[domino.R][domino.L] = true
	}

	return table
}
