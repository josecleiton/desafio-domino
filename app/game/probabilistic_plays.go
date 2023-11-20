package game

import (
	"container/list"
	"fmt"
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
const firstTreeDepth = 1

var tree *guessTree
var treeGeneratingWg sync.WaitGroup

func (g guessTreeGenerate) GeneratePlays(player models.PlayerPosition, hand []models.Domino, parent *guessTreeNode) *list.List {
	result := list.New()

	bonePlay := func(bone models.Domino, edge models.Edge) *guessTreeGenerateStack {
		edgeBone := parent.Table[0]
		if edge == models.RightEdge {
			edgeBone = parent.Table[len(parent.Table)-1]
		}

		glue := dominoInTableFromDomino(edgeBone, edge).Glue(bone)
		if glue == nil {
			return nil
		}

		newTable := make([]models.Domino, len(parent.Table)+1)

		index := 0
		if edge == models.LeftEdge {
			index = 1
		}

		copy(newTable[index:], parent.Table)

		if edge == models.LeftEdge {
			newTable[0] = *glue
		} else {
			newTable[len(newTable)-1] = *glue
		}

		newHand := make([]models.Domino, 0, len(hand)-1)
		for _, v := range hand {
			if v == *glue {
				continue
			}

			newHand = append(newHand, v)
		}

		newPlays := make([]models.DominoPlay, len(g.Plays)+1)
		copy(newPlays, g.Plays)
		newPlays[len(newPlays)-1] = models.DominoPlay{
			PlayerPosition: player,
			Bone: models.DominoInTable{
				Edge:   edge,
				Domino: *glue,
			},
		}

		return &guessTreeGenerateStack{
			generate: guessTreeGenerate{
				Player:   player,
				TableMap: tableMapFromDominoes(newTable),
				Hand:     newHand,
				Plays:    newPlays,
			},
			player:           player,
			unavailableBones: unavailableBones,
			node: &guessTreeNode{
				Player:   player,
				Table:    newTable,
				Parent:   parent,
				Hand:     newHand,
				Depth:    parent.Depth + 1,
				Children: list.New(),
			},
		}
	}

	for _, bone := range hand {
		left, right := bonePlay(bone, models.LeftEdge),
			bonePlay(bone, models.RightEdge)

		if left != nil {
			result.PushBack(left)
		}

		if right != nil {
			result.PushBack(right)
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

func maximizeWinningChancesPlay(
	plays ...*models.DominoPlayWithPass,
) *models.DominoPlayWithPass {
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
					drawLeafDepths = append(
						drawLeafDepths,
						deltaDepth,
					)
					continue
				}

				if !leaf.Winner {
					continue
				}

				winningLeafDepths = append(
					winningLeafDepths,
					deltaDepth,
				)
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

	playWinningTableLen, playDrawTableLen :=
		len(playWinningTable), len(playDrawTable)
	if playWinningTableLen == 0 && playDrawTableLen == 0 {
		return nil
	}

	if playWinningTableLen == 0 {
		return betterPlayFromPlayDepth(plays, playDrawTable)
	}

	return betterPlayFromPlayDepth(plays, playWinningTable)
}

func betterPlayFromPlayDepth(
	plays []*models.DominoPlayWithPass,
	playDepthTable map[*models.DominoPlayWithPass][]int,
) *models.DominoPlayWithPass {
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

func generateTreeByPlay(
	state *models.DominoGameState,
	play *models.DominoPlayWithPass,
) {
	newTableLen := len(state.Table)
	hasNewBone := play.Bone != nil
	if hasNewBone {
		newTableLen++
	}
	newTable := make([]models.Domino, newTableLen)

	{
		index := 0
		if hasNewBone && play.Bone.Edge == models.LeftEdge {
			index = 1
		}

		copy(newTable[index:], state.Table)
	}

	if hasNewBone {
		if play.Bone.Edge == models.LeftEdge {
			newTable[0] = play.Bone.Domino
		} else {
			newTable[len(newTable)-1] = play.Bone.Domino
		}
	}

	newTableMap := make(models.TableMap, models.DominoUniqueBones)
	for _, v := range newTable {
		if _, ok := newTableMap[v.X]; !ok {
			newTableMap[v.X] = make(
				models.TableBone,
				models.DominoUniqueBones,
			)
		}

		if _, ok := newTableMap[v.Y]; !ok {
			newTableMap[v.Y] = make(
				models.TableBone,
				models.DominoUniqueBones,
			)
		}

		newTableMap[v.X][v.Y] = true
		newTableMap[v.Y][v.X] = true
	}

	newHand := make([]models.Domino, 0, len(state.Hand)-1)

	for _, bh := range state.Hand {
		if play.Bone != nil &&
			bh == play.Bone.Domino ||
			bh.Reversed() == play.Bone.Domino {
			continue
		}

		newHand = append(newHand, bh)
	}

	newPlays := make([]models.DominoPlay, 0, len(state.Plays)+1)
	newPlays = append(newPlays, state.Plays...)
	if hasNewBone {
		newPlays = append(newPlays, models.DominoPlay{
			PlayerPosition: play.PlayerPosition,
			Bone:           *play.Bone,
		})
	}

	generateTree(
		&models.DominoGameState{
			PlayerPosition: play.PlayerPosition,
			Hand:           newHand,
			Table:          newTable,
			TableMap:       newTableMap,
			Plays:          newPlays,
		},
		guessTreeGenerate{
			Player:   g.Player,
			Hand:     newHand,
			Plays:    newPlays,
			TableMap: newTableMap,
		},
	)
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

	var unavailableBonesCopy models.UnavailableBonesPlayer

	g.UnavailableBonesMutex.Lock()
	{
		defer g.UnavailableBonesMutex.Unlock()

		delta := models.DominoLength - len(state.Table) +
			len(state.Hand) + len(g.UnavailableBones[state.PlayerPosition.Next()])
		if delta < startGeneratingTreeDelta {
			return
		}

		unavailableBonesCopy = g.UnavailableBones.Copy()
	}

	table := make([]models.Domino, len(state.Table))
	hand := make([]models.Domino, len(state.Hand))

	copy(table, state.Table)
	copy(hand, state.Hand)

	go func() {
		defer treeGeneratingWg.Done()
		tree = new(guessTree)

		node := new(guessTreeNode)
		*node = guessTreeNode{
			Player:   state.PlayerPosition,
			Table:    table,
			Hand:     hand,
			Depth:    firstTreeDepth,
			Children: list.New(),
		}

		tree.Root = node
		tree.Cursor = node

		generateTreePlays(&guessTreeGenerateStack{
			generate:         generate,
			player:           g.Player,
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
		fmt.Println(stack.Len())
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
				Draw: false,
				Winner: top.node.Player == g.Player ||
					top.node.Player == getDuo(),
			})

			continue
		}

		passLeaf := leafFromPasses(top)
		if passLeaf != nil {
			leafPushBack(top, leafs, passLeaf)
			continue
		}

		currentPlayer := top.player.Next()
		{
			foundHand := []models.Domino{}
			if top.node.Depth >= models.DominoMaxPlayer {
				foundHand = searchHand(top.node, currentPlayer)
			}

			if len(foundHand) > 0 {
				childNodes := top.generate.GeneratePlays(
					currentPlayer,
					foundHand,
					top.node,
				)
				stack.PushBackList(generateStackList(
					top,
					childNodes,
					currentPlayer,
				))
				continue
			}
		}

		dominoes := restingDominoes(
			top,
			currentPlayer,
			top.unavailableBones,
		)

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

			childNodes := top.generate.GeneratePlays(
				currentPlayer,
				possibleHand,
				top.node,
			)

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

			newUnavailableBones := top.unavailableBones.Copy()

			newUnavailableBones[currentPlayer][top.node.Table[0].X] = true
			newUnavailableBones[currentPlayer][top.node.Table[len(top.node.Table)-1].Y] = true

			stack.PushBack(&guessTreeGenerateStack{
				player:           currentPlayer,
				generate:         generate,
				unavailableBones: newUnavailableBones,
				node:             top.node.Children.Back().Value.(*guessTreeNode),
			})
		}
		k = 0
	}

	tree.Leafs = leafs

	return tree

}

func searchHand(
	top *guessTreeNode,
	searchPlayer models.PlayerPosition,
) []models.Domino {
	for current := top.Parent; current != nil; current = current.Parent {
		if searchPlayer != current.Player {
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
		childNode := current.Value.(*guessTreeGenerateStack)

		top.node.Children.PushBack(&childNode.node)
		result.PushBack(childNode)

	}

	return result
}

func leafPushBack(
	top *guessTreeGenerateStack,
	leafs *list.List,
	leaf *guessTreeLeaf,
) {
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

	currentCoupleSum := handSumPlayer[g.Player] + handSumPlayer[duo]
	otherCoupleSum := handSumPlayer[g.Player.Next()] + handSumPlayer[duo.Next()]

	if currentCoupleSum < otherCoupleSum || (currentCoupleSum == otherCoupleSum &&
		lastBlockedNode != nil &&
		lastBlockedNode.Player != g.Player &&
		lastBlockedNode.Player != duo) {
		winner = true
	}

	return &guessTreeLeaf{
		guessTreeNode: *top.node,
		Draw:          true,
		Winner:        winner,
	}
}

func restingDominoes(
	top *guessTreeGenerateStack,
	player models.PlayerPosition,
	ub models.UnavailableBonesPlayer,
) []models.Domino {
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

	for boneX, v := range top.generate.TableMap {
		for boneY, ok := range v {
			if !ok {
				continue
			}

			cannotPlayMap[boneX][boneY] = true
			cannotPlayMap[boneY][boneX] = true
		}
	}

	for _, v := range searchOtherHandsDominoes(top, player) {
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

func searchOtherHandsDominoes(
	top *guessTreeGenerateStack,
	player models.PlayerPosition,
) []models.Domino {
	result := make([]models.Domino, 0, models.DominoHandLength*3)

	i := 0
	for current := top.node; current != nil && i < models.DominoMaxPlayer; current = current.Parent {
		i++

		if current.Player == player {
			continue
		}

		result = append(result, current.Hand...)
	}

	return result
}
