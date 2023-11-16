package game

import (
	"sync"

	"github.com/josecleiton/domino/app/models"
	"gonum.org/v1/gonum/stat/combin"
)

type guessTreeNode struct {
	Player   models.PlayerPosition
	Table    []models.Domino
	Children []*guessTreeNode
	Parent   *guessTreeNode
}

type guessTree struct {
	Cursor *guessTreeNode
	Root   *guessTreeNode
	Leafs  []*guessTreeNode
}

type guessTreeGenerate struct {
	TableMap models.TableMap
	Table    []models.Domino
	Hand     []models.Domino
	Plays    []models.DominoPlay
}

type guessTreeGenerateStack struct {
	generate         guessTreeGenerate
	player           models.PlayerPosition
	unavailableBones models.UnavailableBonesPlayer
	node             *guessTreeNode
}

const startGeneratingTreeDelta = 10

var tree *guessTree
var treeGeneratingWg sync.WaitGroup

// func guessPlay(state *models.DominoGameState, left, right []models.DominoInTable) *models.DominoPlayWithPass {
// 	treeGeneratingWg.Wait()
// 	if tree == nil {
// 		return nil
// 	}
// 	return nil
// }

func generateTreeByPlay(state *models.DominoGameState, play *models.DominoPlayWithPass) {
	table := make([]models.Domino, len(state.Table)+1)

	index := 0
	if play.Bone.Edge == models.LeftEdge {
		index = 1
	}

	for i := 0; i < len(state.Table); i++ {
		table[index] = state.Table[i]
		index++
	}

	if play.Bone.Edge == models.LeftEdge {
		table[0] = play.Bone.Domino
	} else {
		table[len(table)-1] = play.Bone.Domino
	}

	hand := make([]models.Domino, 0, len(state.Hand)-1)

	for i := 0; i < len(state.Hand); i++ {
		bh := state.Hand[i]
		if bh == play.Bone.Domino || bh.Reversed() == play.Bone.Domino {
			continue
		}

		hand = append(hand, bh)
	}

	allPlays := make([]models.DominoPlay, 0, len(state.Plays))
	allPlays = append(allPlays, state.Plays...)

	generateTree(state, guessTreeGenerate{
		Table: table,
		Hand:  hand,
		Plays: allPlays,
	})
}

func generateTree(state *models.DominoGameState, generate guessTreeGenerate) {
	unavailableBonesCopy := make(models.UnavailableBonesPlayer, models.DominoMaxPlayer)
	for player, tableMap := range unavailableBones {
		for boneSide, v := range tableMap {
			if !v {
				continue
			}

			if _, ok := unavailableBonesCopy[player]; !ok {
				unavailableBonesCopy[player] = make(models.TableBone, models.DominoUniqueBones)
			}

			unavailableBonesCopy[player][boneSide] = true
		}
	}

	nextPlayer := (state.PlayerPosition + 1) % models.DominoMaxPlayer
	if len(state.Table)+len(state.Hand)+len(unavailableBonesCopy[nextPlayer]) < startGeneratingTreeDelta {
		return
	}

	treeGeneratingWg.Add(1)
	go func() {
		defer treeGeneratingWg.Done()
		tree = new(guessTree)

		node := new(guessTreeNode)
		node.Player = state.PlayerPosition
		node.Table = generate.Table

		tree.Root = node
		tree.Cursor = node

		children, leafs := generateTreePlays(&guessTreeGenerateStack{
			generate:         generate,
			player:           player.Add(1),
			unavailableBones: unavailableBonesCopy,
			node:             node,
		})

		tree.Root.Children = children
		tree.Leafs = leafs
	}()
}

func generateTreePlays(init *guessTreeGenerateStack) ([]*guessTreeNode, []*guessTreeNode) {
	if init == nil {
		return []*guessTreeNode{}, []*guessTreeNode{}
	}

	stack := make([]*guessTreeGenerateStack, 1, combin.Binomial(startGeneratingTreeDelta, startGeneratingTreeDelta/2-1))
	stack[0] = init

	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		dominoes := restingDominoes(top.generate, top.player, top.unavailableBones)

		currentPlayerPlays := make([]models.DominoPlay, 0, len(top.generate.Plays))
		for _, p := range top.generate.Plays {
			if p.PlayerPosition != top.player {
				continue
			}
			currentPlayerPlays = append(currentPlayerPlays, p)
		}

		n := len(dominoes)
		k := models.DominoHandLength - len(currentPlayerPlays)

		top.node.Children = make([]*guessTreeNode, 0, combin.Binomial(n, k))

		storedIdx := make([]int, k)
		combinationGen := combin.NewCombinationGenerator(n, k)
		for combinationGen.Next() {
			cs := combinationGen.Combination(storedIdx)

			possibleHand := make([]models.Domino, 0, len(cs))
			for _, idx := range cs {
				possibleHand = append(possibleHand, dominoes[idx])
			}

			childNode := &guessTreeNode{
				Player: top.player.Add(1),
				Table:  top.generate.Table,
				Parent: top.node,
			}

			stack = append(stack, &guessTreeGenerateStack{
				player: top.player,
				generate: guessTreeGenerate{
					Hand: possibleHand,
				},
				node: childNode,
			})

			top.node.Children = append(top.node.Children, childNode)
		}

	}

	return []*guessTreeNode{}, []*guessTreeNode{}

}

func restingDominoes(generate guessTreeGenerate, player models.PlayerPosition, ub models.UnavailableBonesPlayer) []models.Domino {
	result := make(models.TableMap, models.DominoUniqueBones)

	for boneSide, ok := range ub[player] {
		if !ok {
			continue
		}

		result[boneSide] = make(models.TableBone, models.DominoUniqueBones)
		for i := models.DominoMinBone; i < models.DominoMaxBone; i++ {
			result[boneSide][i] = true
			result[i][boneSide] = true
		}
	}

	for boneX, v := range generate.TableMap {
		for boneY, ok := range v {
			if !ok {
				continue
			}

			if _, ok := result[boneX]; !ok {
				result[boneX] = make(models.TableBone, models.DominoUniqueBones)
			}

			if _, ok := result[boneY]; !ok {
				result[boneY] = make(models.TableBone, models.DominoUniqueBones)
			}

			result[boneX][boneY] = true
			result[boneY][boneX] = true
		}
	}

	dominoes := make([]models.Domino, 0, models.DominoLength)
	limit := models.DominoMaxBone
	for i := models.DominoMinBone; i < limit; i++ {
		for j := i; j < limit; j++ {
			if tb, ok := result[i]; ok {
				if unavailable, ok := tb[j]; ok && unavailable {
					continue
				}
			}

			dominoes = append(dominoes, models.Domino{X: i, Y: j})
		}
	}
	return dominoes
}
