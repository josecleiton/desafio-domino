package game

import (
	"container/list"
	"log"
	"math/rand"
	"reflect"
	"sort"
	"sync"

	"github.com/josecleiton/domino/app/models"
	"gonum.org/v1/gonum/stat/combin"
)

type guessTreeNode struct {
	Player   models.PlayerPosition
	Table    []models.Domino
	Hand     []models.Domino
	Children *list.List
	Parent   *guessTreeNode
	Depth    int
}

type guessTreeLeaf struct {
	guessTreeNode
	Draw   bool
	Winner bool
}

type guessTreeNodeDominoInTable struct {
	guessTreeNode
	bone models.DominoInTable
}

type guessTree struct {
	Cursor *guessTreeNode
	Root   *guessTreeNode
	Leafs  *list.List
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

const startGeneratingTreeDelta = 12

var tree *guessTree
var treeGeneratingWg sync.WaitGroup

func (g guessTreeGenerate) GeneratePlays(player models.PlayerPosition, hand []models.Domino, parent *guessTreeNode) *list.List {
	result := list.New()

	for _, bone := range hand {
		firstBone, lastBone := parent.Table[0],
			parent.Table[len(parent.Table)-1]

		if glue := dominoInTableFromDomino(firstBone, models.LeftEdge).Glue(bone); glue != nil {
			newTable := make([]models.Domino, len(parent.Table)+1)
			copy(newTable[1:], parent.Table)
			newTable[0] = *glue

			newHand := make([]models.Domino, 0, len(hand)-1)
			for _, v := range hand {
				if v == *glue {
					continue
				}

				newHand = append(newHand, v)
			}

			result.PushBack(&guessTreeNodeDominoInTable{
				guessTreeNode: guessTreeNode{
					Player:   player,
					Table:    newTable,
					Parent:   parent,
					Hand:     newHand,
					Depth:    parent.Depth + 1,
					Children: list.New(),
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

			result.PushBack(&guessTreeNodeDominoInTable{
				guessTreeNode: guessTreeNode{
					Player:   player,
					Table:    newTable,
					Parent:   parent,
					Hand:     hand,
					Depth:    parent.Depth + 1,
					Children: list.New(),
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

		for current := node.Children.Front(); current != nil; current = current.Next() {
			queue.PushBack(current.Value.(*guessTreeNode))
		}
	}

	log.Println("missing play on tree")

	return t.Cursor
}

func maximizeWinningChancesPlay(plays ...*models.DominoPlayWithPass) *models.DominoPlayWithPass {
	if tree == nil || len(plays) == 0 {
		return nil
	}

	treeGeneratingWg.Wait()

	treeTraversingWg := sync.WaitGroup{}
	treeTraversingWg.Add(len(plays))

	playWinningTable := make(map[*models.DominoPlayWithPass][]int, len(plays))
	playDrawTable := make(map[*models.DominoPlayWithPass][]int, len(plays))

	for _, play := range plays {
		playWinningTable[play] = []int{}
		playDrawTable[play] = []int{}
		go func(play *models.DominoPlayWithPass) {
			defer treeTraversingWg.Done()
			winningLeafDepths := playWinningTable[play]
			drawLeafDepths := playDrawTable[play]

			for current := tree.Leafs.Front(); current != nil; current = current.Next() {
				leaf := current.Value.(*guessTreeLeaf)
				deltaDepth := leaf.Depth - tree.Cursor.Depth

				if leaf.Draw && leaf.Winner {
					drawLeafDepths = append(drawLeafDepths, deltaDepth)
					continue
				}

				if !leaf.Winner {
					continue
				}

				winningLeafDepths = append(winningLeafDepths, deltaDepth)
			}

			sort.Slice(winningLeafDepths, func(i, j int) bool {
				return winningLeafDepths[i] < winningLeafDepths[j]
			})

			sort.Slice(drawLeafDepths, func(i, j int) bool {
				return drawLeafDepths[i] < drawLeafDepths[j]
			})
		}(play)
	}

	treeTraversingWg.Wait()

	for k, v := range playWinningTable {
		if len(v) > 0 {
			continue
		}

		delete(playWinningTable, k)
	}

	for k, v := range playDrawTable {
		if len(v) > 0 {
			continue
		}
		delete(playDrawTable, k)
	}

	playWinningTableLen, playDrawTableLen := len(playWinningTable), len(playDrawTable)
	if playWinningTableLen == 0 && playDrawTableLen == 0 {
		return nil
	}

	if playWinningTableLen == 0 {
		return betterPlayFromPlayDepth(plays, playDrawTable)
	}

	return betterPlayFromPlayDepth(plays, playWinningTable)
}

func betterPlayFromPlayDepth(plays []*models.DominoPlayWithPass, playDepthTable map[*models.DominoPlayWithPass][]int) *models.DominoPlayWithPass {
	if len(playDepthTable) == 0 {
		return nil
	}

	betterPlay := plays[0]
	for _, play := range plays[1:] {
		if playDepthTable[betterPlay][0] <= 1 {
			return betterPlay
		}

		if playDepthTable[play][0] <= 1 {
			return play
		}

		if len(playDepthTable[betterPlay]) <= len(playDepthTable[play]) {
			continue
		}

		betterPlay = play
	}

	return betterPlay
}

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
		*node = guessTreeNode{
			Player:   state.PlayerPosition,
			Table:    table,
			Hand:     state.Hand,
			Depth:    0,
			Children: list.New(),
		}

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

	stack := list.New()
	stack.PushBack(init)

	leafs := list.New()

	for stack.Len() > 0 {
		element := stack.Back()
		top := element.Value.(*guessTreeGenerateStack)
		stack.Remove(element)

		if len(top.node.Table) == models.DominoLength {
			leafPushBack(top, leafs, &guessTreeLeaf{
				guessTreeNode: guessTreeNode{
					Player: top.player,
					Hand:   top.node.Hand,
					Table:  top.node.Table,
					Parent: top.node,
					Depth:  top.node.Depth + 1,
				},
				Draw:   false,
				Winner: top.node.Player == player || top.node.Player == getDuo(),
			})

			continue
		}

		passLeaf := leafFromPasses(top)
		if passLeaf != nil {
			leafPushBack(top, leafs, passLeaf)
			continue
		}

		currentPlayer := top.player.Add(1)

		{
			hand = []models.Domino{}
			if top.node.Depth > models.DominoMaxPlayer-1 {
				hand = searchHand(top.node)
			}

			if len(hand) > 0 {
				childNodes := top.generate.GeneratePlays(currentPlayer, hand, top.node)
				stack.PushBackList(generateStackList(
					top,
					childNodes,
					currentPlayer,
				))
				continue
			}
		}

		dominoes := restingDominoes(top.generate, currentPlayer, top.unavailableBones)

		playerPlaysLen := 0
		for _, p := range top.generate.Plays {
			if p.PlayerPosition != currentPlayer {
				continue
			}
			playerPlaysLen++
		}

		n := len(dominoes)
		k := models.DominoHandLength - playerPlaysLen

		if n < k || k == 0 {
			continue
		}

		storedIdx := make([]int, k)
		combinationGen := combin.NewCombinationGenerator(n, k)

		log.Println("generating children:", combin.Binomial(n, k))

		for combinationGen.Next() {
			cs := combinationGen.Combination(storedIdx)

			possibleHand := make([]models.Domino, 0, len(cs))
			for _, idx := range cs {
				possibleHand = append(possibleHand, dominoes[idx])
			}

			childNodes := top.generate.GeneratePlays(currentPlayer, possibleHand, top.node)

			if childNodes.Len() > 0 {
				stack.PushBackList(generateStackList(
					top,
					childNodes,
					currentPlayer,
				))
				continue
			}

			// player passed
			top.node.Children.PushBack(&guessTreeNode{
				Player:   currentPlayer,
				Table:    top.node.Table,
				Parent:   top.node,
				Hand:     possibleHand,
				Depth:    top.node.Depth + 1,
				Children: list.New(),
			})
			generate := guessTreeGenerate{
				Hand:     possibleHand,
				TableMap: top.generate.TableMap,
				Plays:    top.generate.Plays,
				Player:   currentPlayer,
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

			stack.PushBack(&guessTreeGenerateStack{
				player:           currentPlayer,
				generate:         generate,
				unavailableBones: unavailableBones,
				node:             top.node.Children.Back().Value.(*guessTreeNode),
			})
		}
		k = 0
	}

	tree.Leafs = leafs

	return tree

}

func searchHand(top *guessTreeNode) []models.Domino {
	for current := top.Parent; current != nil; current = current.Parent {
		if top.Player != current.Player {
			continue
		}

		hand := make([]models.Domino, len(current.Hand))
		copy(hand, current.Hand)

		return hand
	}

	return []models.Domino{}
}

func generateStackList(
	top *guessTreeGenerateStack,
	childNodes *list.List,
	currentPlayer models.PlayerPosition,
) *list.List {
	result := list.New()
	for current := childNodes.Front(); current != nil; current = current.Next() {
		childNode := current.Value.(*guessTreeNodeDominoInTable)

		top.node.Children.PushBack(&childNode.guessTreeNode)

		possiblePlays := make([]models.DominoPlay, 0, len(top.generate.Plays)+1)
		possiblePlays = append(possiblePlays, top.generate.Plays...)
		possiblePlays = append(possiblePlays, models.DominoPlay{
			PlayerPosition: currentPlayer,
			Bone:           childNode.bone,
		})

		possibleTableMap := make(models.TableMap, models.DominoUniqueBones)
		for _, play := range possiblePlays {
			if _, ok := possibleTableMap[play.Bone.X]; !ok {
				possibleTableMap[play.Bone.X] = make(models.TableBone, models.DominoUniqueBones)
			}

			if _, ok := possibleTableMap[play.Bone.Y]; !ok {
				possibleTableMap[play.Bone.Y] = make(models.TableBone, models.DominoUniqueBones)
			}

			possibleTableMap[play.Bone.X][play.Bone.Y] = true
			possibleTableMap[play.Bone.Y][play.Bone.X] = true
		}

		top.node.Children.PushBack(&childNode.guessTreeNode)

		result.PushBack(&guessTreeGenerateStack{
			player:           currentPlayer,
			unavailableBones: top.unavailableBones,
			generate: guessTreeGenerate{
				Hand:     childNode.Hand,
				TableMap: possibleTableMap,
				Plays:    possiblePlays,
				Player:   currentPlayer,
			},
			node: top.node.Children.Back().Value.(*guessTreeNode),
		})

	}

	return result
}

func leafPushBack(top *guessTreeGenerateStack, leafs *list.List, leaf *guessTreeLeaf) {
	parent := top.node.Parent

	for current := top.node.Children.Front(); current != nil; current = current.Next() {
		currentValue := current.Value.(*guessTreeNode)
		if currentValue == top.node {
			top.node.Children.Remove(current)
			break
		}
	}

	newLeaf := &guessTreeLeaf{
		guessTreeNode: guessTreeNode{
			Player:   leaf.Player,
			Table:    leaf.Table,
			Hand:     parent.Hand,
			Parent:   parent,
			Depth:    parent.Depth + 1,
			Children: list.New(),
		},
		Draw:   leaf.Draw,
		Winner: leaf.Winner,
	}

	parent.Children.PushBack(&newLeaf.guessTreeNode)
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
		Draw:          true,
		Winner:        winner,
	}
}

func restingDominoes(generate guessTreeGenerate, player models.PlayerPosition, ub models.UnavailableBonesPlayer) []models.Domino {
	cannotPlayMap := make(models.TableMap, models.DominoUniqueBones)

	const maxBone = models.DominoMaxBone
	for i := models.DominoMinBone; i <= maxBone; i++ {
		cannotPlayMap[i] = make(models.TableBone, models.DominoUniqueBones)
	}

	for boneSide, ok := range ub[player] {
		if !ok {
			continue
		}

		for i := models.DominoMinBone; i <= models.DominoMaxBone; i++ {
			cannotPlayMap[boneSide][i] = true
			cannotPlayMap[i][boneSide] = true
		}
	}

	for boneX, v := range generate.TableMap {
		for boneY, ok := range v {
			if !ok {
				continue
			}

			cannotPlayMap[boneX][boneY] = true
			cannotPlayMap[boneY][boneX] = true
		}
	}

	for _, v := range generate.Hand {
		cannotPlayMap[v.X][v.Y] = true
		cannotPlayMap[v.Y][v.X] = true
	}

	dominoes := make([]models.Domino, 0, startGeneratingTreeDelta)
	for i := models.DominoMinBone; i <= maxBone; i++ {
		for j := i; j <= maxBone; j++ {
			if unavailable, ok := cannotPlayMap[i][j]; ok && unavailable {
				continue
			}

			dominoes = append(dominoes, models.Domino{X: i, Y: j})
		}
	}

	rand.Shuffle(len(dominoes), func(i, j int) {
		dominoes[i], dominoes[j] = dominoes[j], dominoes[i]
	})

	return dominoes
}
