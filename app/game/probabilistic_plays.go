package game

import (
	"container/list"
	"log"
	"math/rand"
	"reflect"
	"sync"

	"github.com/josecleiton/domino/app/models"
	"gonum.org/v1/gonum/stat/combin"
)

type guessTreeNode struct {
	Player   models.PlayerPosition
	Table    []models.Domino
	Hand     []models.Domino
	Children []*guessTreeNode
	Parent   *guessTreeNode
}

type guessTreeLeaf struct {
	guessTreeNode
	draw   bool
	winner bool
}

type guessTreeNodeDominoInTable struct {
	guessTreeNode
	bone models.DominoInTable
}

type guessTree struct {
	Cursor *guessTreeNode
	Root   *guessTreeNode
	Leafs  []*guessTreeLeaf
}

type guessTreeGenerate struct {
	Player   models.PlayerPosition
	TableMap models.TableMap
	Hand     []models.Domino
	Plays    []models.DominoPlay
}

type guessTreeGenerateStack struct {
	generate         guessTreeGenerate
	player           models.PlayerPosition
	unavailableBones models.UnavailableBonesPlayer
	node             *guessTreeNode
}

const startGeneratingTreeDelta = 8

var tree *guessTree
var treeGeneratingWg sync.WaitGroup

func (g guessTreeGenerate) GeneratePlays(player models.PlayerPosition, hand []models.Domino, parent *guessTreeNode) []*guessTreeNodeDominoInTable {
	result := make([]*guessTreeNodeDominoInTable, 0, len(hand))

	for _, bone := range hand {
		firstBone, lastBone := parent.Table[0], parent.Table[len(parent.Table)-1]
		if glue := dominoInTableFromDomino(firstBone, models.LeftEdge).Glue(bone); glue != nil {
			newTable := make([]models.Domino, len(parent.Table)+1)
			for i := 1; i < len(newTable); i++ {
				newTable[i] = parent.Table[i-1]
			}
			newTable[0] = *glue

			result = append(result, &guessTreeNodeDominoInTable{
				guessTreeNode: guessTreeNode{
					Player: player,
					Table:  newTable,
					Parent: parent,
					Hand:   hand,
				},
				bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: *glue,
				},
			})
		}

		if glue := dominoInTableFromDomino(lastBone, models.RightEdge).Glue(bone); glue != nil {
			newTable := make([]models.Domino, len(parent.Table)+1)
			copy(newTable, parent.Table)

			newTable[len(newTable)-1] = *glue

			result = append(result, &guessTreeNodeDominoInTable{
				guessTreeNode: guessTreeNode{
					Player: player,
					Table:  newTable,
					Parent: parent,
					Hand:   hand,
				},
				bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: *glue,
				},
			})
		}
	}

	return result
}

func (t guessTree) RepositionCursor(generate guessTreeGenerate) *guessTreeNode {
	queue := list.New()
	queue.PushBack(t.Cursor)

	for queue.Len() > 0 {
		e := queue.Front()
		node := e.Value.(*guessTreeNode)

		queue.Remove(e)

		if node.Player == generate.Player &&
			reflect.DeepEqual(node.Hand, generate.Hand) &&
			reflect.DeepEqual(node.Table, generate.TableMap) {
			return node
		}

		for _, child := range node.Children {
			queue.PushBack(child)
		}
	}

	log.Println("missing play on tree")

	return t.Cursor
}

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

	tableMap := make(models.TableMap, models.DominoUniqueBones)
	for _, v := range table {
		if _, ok := tableMap[v.X]; !ok {
			tableMap[v.X] = make(models.TableBone, models.DominoUniqueBones)
		}

		if _, ok := tableMap[v.Y]; !ok {
			tableMap[v.Y] = make(models.TableBone, models.DominoUniqueBones)
		}

		tableMap[v.X][v.Y] = true
		tableMap[v.Y][v.X] = true
	}

	hand := make([]models.Domino, 0, len(state.Hand)-1)

	for i := 0; i < len(state.Hand); i++ {
		bh := state.Hand[i]
		if bh == play.Bone.Domino || bh.Reversed() == play.Bone.Domino {
			continue
		}

		hand = append(hand, bh)
	}

	allPlays := make([]models.DominoPlay, 0, len(state.Plays)+1)
	allPlays = append(allPlays, state.Plays...)
	if play.Bone != nil {
		allPlays = append(allPlays, models.DominoPlay{
			PlayerPosition: play.PlayerPosition,
			Bone:           *play.Bone,
		})
	}

	generateTree(state, guessTreeGenerate{
		Player:   player,
		Hand:     hand,
		Plays:    allPlays,
		TableMap: tableMap,
	})
}

func generateTree(state *models.DominoGameState, generate guessTreeGenerate) {
	treeGeneratingWg.Add(1)
	if tree != nil {
		go func() {
			defer treeGeneratingWg.Done()
			tree.Cursor = tree.RepositionCursor(generate)
		}()
		return
	}

	nextPlayer := state.PlayerPosition.Add(1)
	unavailableBonesCopy := make(models.UnavailableBonesPlayer, models.DominoMaxPlayer)

	unavailableBonesMutex.Lock()
	{
		defer unavailableBonesMutex.Unlock()
		if models.DominoLength-len(state.Table)+len(state.Hand)+len(unavailableBones[nextPlayer]) < startGeneratingTreeDelta {
			return
		}

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
	}

	table := make([]models.Domino, len(state.Table))
	table = append(table, state.Table...)

	go func() {
		defer treeGeneratingWg.Done()
		tree = new(guessTree)

		node := new(guessTreeNode)
		node.Player = state.PlayerPosition
		node.Table = table
		node.Hand = state.Hand

		tree.Root = node
		tree.Cursor = node

		generateTreePlays(&guessTreeGenerateStack{
			generate:         generate,
			player:           player,
			unavailableBones: unavailableBonesCopy,
			node:             node,
		})
	}()
}

func generateTreePlays(init *guessTreeGenerateStack) *guessTree {
	if init == nil {
		return tree
	}

	stack := make([]*guessTreeGenerateStack, 1, combin.Binomial(startGeneratingTreeDelta, startGeneratingTreeDelta/2-1))
	stack[0] = init

	leafs := make([]*guessTreeLeaf, 0, models.DominoLength)

	for len(stack) > 0 {
		top := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if len(top.node.Table) == models.DominoLength {
			leafs = append(leafs, &guessTreeLeaf{
				guessTreeNode: *top.node,
				draw:          false,
				winner:        top.node.Player == player || top.node.Player == getDuo(),
			})
		}

		passLeaf := leafFromPasses(top)
		if passLeaf != nil {
			leafs = append(leafs, passLeaf)
			continue
		}

		currentPlayer := top.player.Add(1)

		dominoes := restingDominoes(top.generate, currentPlayer, top.unavailableBones)

		currentPlayerPlays := make([]models.DominoPlay, 0, len(top.generate.Plays))
		for _, p := range top.generate.Plays {
			if p.PlayerPosition != currentPlayer {
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

			childNodes := top.generate.GeneratePlays(currentPlayer, possibleHand, top.node)

			// player passed
			if len(childNodes) == 0 {
				top.node.Children = append(top.node.Children, &guessTreeNode{
					Player: currentPlayer,
					Table:  top.node.Table,
					Parent: top.node,
					Hand:   possibleHand,
				})
				generate := guessTreeGenerate{
					Hand:     possibleHand,
					TableMap: top.generate.TableMap,
					Plays:    top.generate.Plays,
				}

				unavailableBones := make(models.UnavailableBonesPlayer, models.DominoMaxPlayer)
				if _, ok := unavailableBones[currentPlayer]; !ok {
					unavailableBones[currentPlayer] = make(models.TableBone, models.DominoUniqueBones)
				}
				for k, v := range top.unavailableBones[currentPlayer] {
					if !v {
						continue
					}

					unavailableBones[currentPlayer][k] = true
				}

				unavailableBones[currentPlayer][top.node.Table[0].X] = true
				unavailableBones[currentPlayer][top.node.Table[len(top.node.Table)-1].Y] = true

				stack = append(stack, &guessTreeGenerateStack{
					player:           currentPlayer,
					generate:         generate,
					unavailableBones: unavailableBones,
					node:             top.node.Children[len(top.node.Children)-1],
				})

				continue
			}

			for _, childNode := range childNodes {
				top.node.Children = append(top.node.Children, &childNode.guessTreeNode)

				possiblePlays := make([]models.DominoPlay, 0, len(top.generate.Plays)+1)
				possiblePlays = append(possiblePlays, top.generate.Plays...)
				possiblePlays = append(possiblePlays, models.DominoPlay{
					PlayerPosition: currentPlayer,
					Bone:           childNode.bone,
				})

				possibleTableMap := make(models.TableMap, models.DominoUniqueBones)
				for bx, tb := range top.generate.TableMap {
					if _, ok := possibleTableMap[bx]; !ok {
						possibleTableMap[bx] = make(models.TableBone, models.DominoUniqueBones)
					}

					for by, v := range tb {
						possibleTableMap[bx][by] = v
					}
				}

				stack = append(stack, &guessTreeGenerateStack{
					player: currentPlayer,
					generate: guessTreeGenerate{
						Hand:     possibleHand,
						TableMap: possibleTableMap,
						Plays:    possiblePlays,
					},
					node: &childNode.guessTreeNode,
				})

			}

		}

	}

	tree.Leafs = leafs

	return tree

}

func leafFromPasses(top *guessTreeGenerateStack) *guessTreeLeaf {
	aux := top.node
	passes := 0
	handSumPlayer := make(map[models.PlayerPosition]int, models.DominoMaxPlayer)
	lastBlockedNode := aux
	for aux != nil {
		if aux.Parent != nil && reflect.DeepEqual(aux.Table, aux.Parent.Table) {
			passes++
			for _, b := range aux.Hand {
				handSumPlayer[aux.Player] += b.Sum()
			}
		} else {
			lastBlockedNode = aux
			break
		}
		aux = aux.Parent
	}

	if passes != models.DominoMaxPlayer {
		return nil
	}

	winner := false
	duo := getDuo()

	currentCoupleSum := handSumPlayer[player] + handSumPlayer[duo]
	otherCoupleSum := handSumPlayer[player.Next()] + handSumPlayer[duo.Next()]

	if currentCoupleSum < otherCoupleSum || (currentCoupleSum == otherCoupleSum &&
		lastBlockedNode != nil &&
		lastBlockedNode.Player != player &&
		lastBlockedNode.Player != duo) {
		winner = true
	}

	return &guessTreeLeaf{
		guessTreeNode: *top.node,
		draw:          true,
		winner:        winner,
	}
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

	rand.Shuffle(len(dominoes), func(i, j int) {
		dominoes[i], dominoes[j] = dominoes[j], dominoes[i]
	})

	return dominoes
}
