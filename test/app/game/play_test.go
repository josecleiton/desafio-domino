package game

import (
	"container/list"
	"fmt"
	"testing"

	"github.com/josecleiton/domino/app/game"
	"github.com/josecleiton/domino/app/models"
)

func firstPlay() models.DominoGameState {
	return models.DominoGameState{
		PlayerPosition: 1,
		Hand: []models.Domino{
			{L: 0, R: 0},
			{L: 0, R: 3},
			{L: 1, R: 2},
			{L: 1, R: 3},
			{L: 1, R: 6},
			{L: 3, R: 6},
			{L: 5, R: 5},
		},
		TableMap: models.TableMap{},
		Table:    []models.Domino{},
		Plays:    []models.DominoPlay{},
	}
}

func secondPlay(stGameState *models.DominoGameState, stPlay *models.DominoPlayWithPass) models.DominoGameState {
	plays := []models.DominoPlay{
		{
			PlayerPosition: stPlay.PlayerPosition,
			Bone: models.DominoInTable{
				Edge:   stPlay.Bone.Edge,
				Domino: stPlay.Bone.Domino,
			},
		},
		{
			PlayerPosition: stPlay.PlayerPosition + 1,
			Bone: models.DominoInTable{
				Edge: models.LeftEdge,
				Domino: models.Domino{
					L: 4,
					R: 5,
				},
			},
		},
		{
			PlayerPosition: stPlay.PlayerPosition + 2,
			Bone: models.DominoInTable{
				Edge: models.RightEdge,
				Domino: models.Domino{
					L: 5,
					R: 3,
				},
			},
		},
	}

	table := []models.Domino{
		plays[1].Bone.Domino,
		plays[0].Bone.Domino,
		plays[2].Bone.Domino,
	}

	tableMap := make(models.TableMap, len(plays))
	for _, play := range plays {
		if _, ok := tableMap[play.Bone.L]; !ok {
			tableMap[play.Bone.L] = make(models.TableBone, len(plays))
		}

		if _, ok := tableMap[play.Bone.R]; !ok {
			tableMap[play.Bone.R] = make(models.TableBone, len(plays))
		}

		tableMap[play.Bone.L][play.Bone.R] = true
		tableMap[play.Bone.R][play.Bone.L] = true
	}

	newHand := make([]models.Domino, 0, len(stGameState.Hand)-1)
	for _, bone := range stGameState.Hand {
		if bone != stPlay.Bone.Domino {
			newHand = append(newHand, bone)
		}
	}

	return models.DominoGameState{
		PlayerPosition: stPlay.PlayerPosition,
		Hand:           newHand,
		TableMap:       tableMap,
		Plays:          plays,
		Table:          table,
	}
}

func TestPlayInitial(t *testing.T) {
	firstPlay := firstPlay()
	if len(firstPlay.Hand) < 1 {
		t.Fatal("Hand is not empty")
	}

	play := game.Play(&firstPlay)

	if play.Pass() {
		t.Fatal("Pass is not allowed on first play")
	}

	maxBone := firstPlay.Hand[0]
	for _, bone := range firstPlay.Hand[1:] {
		if bone.Sum() > maxBone.Sum() {
			maxBone = bone
		}
	}

	if play.Bone.Sum() != maxBone.Sum() {
		t.Fatal("Not maximized play")
	}

}

func TestPlayGlue(t *testing.T) {
	stGameState := firstPlay()
	stPlay := game.Play(&stGameState)
	ndGameState := secondPlay(&stGameState, &stPlay)

	fmt.Println("hand:", ndGameState.Hand)
	fmt.Println("table:", ndGameState.Table)
	ndPlay := game.Play(&ndGameState)
	if ndPlay.Pass() {
		fmt.Println("Pass is not allowed")
		t.FailNow()
	}

	fromHand := false
	for _, bone := range ndGameState.Hand {
		if bone == ndPlay.Bone.Domino || bone.Reversed() == ndPlay.Bone.Domino {
			fromHand = true
		}
	}

	if !fromHand {
		t.Errorf("Bone %v not found in hand %v\n", ndPlay.Bone.Domino, ndGameState.Hand)
	}

	fmt.Println("newPlay:", ndPlay)
}

func TestPassedPlay(t *testing.T) {
	tableSt := []models.Domino{
		{L: 0, R: 2},
		{L: 2, R: 5},
		{L: 5, R: 5},
		{L: 5, R: 6},
		{L: 6, R: 6},
		{L: 6, R: 0},
		{L: 0, R: 3},
		{L: 3, R: 3},
		{L: 3, R: 6},
	}
	tableMapSt := tableMapFromTable(tableSt)
	gameStateSt := &models.DominoGameState{
		PlayerPosition: 2,
		Hand: []models.Domino{
			{L: 6, R: 1},
			{L: 5, R: 3},
			{L: 1, R: 0},
			{L: 0, R: 0},
			{L: 4, R: 3},
		},
		Plays: []models.DominoPlay{
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 6, R: 6},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 5, R: 6},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 6, R: 0},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 5, R: 5},
				},
			},
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 2, R: 5},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 0, R: 3},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 0, R: 2},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 3, R: 3},
				},
			},
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 3, R: 6},
				},
			},
		},
		Table:    tableSt,
		TableMap: tableMapSt,
	}

	play := game.Play(gameStateSt)
	if play.Pass() {
		t.Fatal("Pass is not allowed")
	}

	fmt.Println("play:", play)

	tableNd := []models.Domino{
		{L: 5, R: 0},
		{L: 0, R: 2},
		{L: 2, R: 5},
		{L: 5, R: 5},
		{L: 5, R: 6},
		{L: 6, R: 6},
		{L: 6, R: 0},
		{L: 0, R: 3},
		{L: 3, R: 3},
		{L: 3, R: 6},
		{L: 6, R: 1},
		{L: 1, R: 3},
	}
	tableMapNd := tableMapFromTable(tableNd)

	gameStateNd := &models.DominoGameState{
		PlayerPosition: 2,
		Hand: []models.Domino{
			{L: 5, R: 3},
			{L: 1, R: 0},
			{L: 0, R: 0},
			{L: 4, R: 3},
		},
		Plays: []models.DominoPlay{
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 6, R: 6},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 5, R: 6},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 6, R: 0},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 5, R: 5},
				},
			},
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 2, R: 5},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 0, R: 3},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 0, R: 2},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 3, R: 3},
				},
			},
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 3, R: 6},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 6, R: 1},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{L: 1, R: 3},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{L: 5, R: 0},
				},
			},
		},
		Table:    tableNd,
		TableMap: tableMapNd,
	}

	play = game.Play(gameStateNd)

	fmt.Println("play:", play)

	if (play.Bone.Domino != models.Domino{L: 3, R: 5}) {
		t.Fatal("Wrong play")
	}
}

func TestGenerateTree(t *testing.T) {
	plays := []models.DominoPlay{
		{
			PlayerPosition: 1,
			Bone: models.DominoInTable{
				Edge:   models.LeftEdge,
				Domino: models.Domino{L: 6, R: 6},
			},
		},
		{
			PlayerPosition: 2,
			Bone: models.DominoInTable{
				Edge:   models.LeftEdge,
				Domino: models.Domino{L: 1, R: 6},
			},
		},
		{
			PlayerPosition: 3,
			Bone: models.DominoInTable{
				Edge:   models.LeftEdge,
				Domino: models.Domino{L: 0, R: 1},
			},
		},
		{
			PlayerPosition: 4,
			Bone: models.DominoInTable{
				Edge:   models.RightEdge,
				Domino: models.Domino{L: 6, R: 2},
			},
		},
		{
			PlayerPosition: 1,
			Bone: models.DominoInTable{
				Edge:   models.LeftEdge,
				Domino: models.Domino{L: 0, R: 0},
			},
		},
		{
			PlayerPosition: 2,
			Bone: models.DominoInTable{
				Edge:   models.LeftEdge,
				Domino: models.Domino{L: 3, R: 0},
			},
		},
		{
			PlayerPosition: 3,
			Bone: models.DominoInTable{
				Edge:   models.LeftEdge,
				Domino: models.Domino{L: 5, R: 3},
			},
		},
		{
			PlayerPosition: 4,
			Bone: models.DominoInTable{
				Edge:   models.RightEdge,
				Domino: models.Domino{L: 2, R: 2},
			},
		},
		{
			PlayerPosition: 1,
			Bone: models.DominoInTable{
				Edge:   models.LeftEdge,
				Domino: models.Domino{L: 5, R: 5},
			},
		},
		{
			PlayerPosition: 2,
			Bone: models.DominoInTable{
				Edge:   models.RightEdge,
				Domino: models.Domino{L: 2, R: 1},
			},
		},
		{
			PlayerPosition: 3,
			Bone: models.DominoInTable{
				Edge:   models.LeftEdge,
				Domino: models.Domino{L: 0, R: 5},
			},
		},
		{
			PlayerPosition: 4,
			Bone: models.DominoInTable{
				Edge:   models.LeftEdge,
				Domino: models.Domino{L: 6, R: 0},
			},
		},
	}
	tableL := list.New()

	for _, p := range plays {
		if p.Bone.Edge == models.LeftEdge {
			tableL.PushFront(p.Bone.Domino)
		} else {
			tableL.PushBack(p.Bone.Domino)
		}
	}

	table := make([]models.Domino, 0, tableL.Len())
	for current := tableL.Front(); current != nil; current = current.Next() {
		table = append(table, current.Value.(models.Domino))
	}

	tableMap := tableMapFromTable(table)

	state := &models.DominoGameState{
		PlayerPosition: 1,
		Hand: []models.Domino{
			{L: 6, R: 5},
			{L: 1, R: 1},
			{L: 3, R: 1},
			{L: 4, R: 1},
		},
		Table:    table,
		TableMap: tableMap,
		Plays:    plays,
	}
	play := game.Play(state)

	if play.Pass() {
		t.Error("Pass is not allowed")
	}
}

func tableMapFromTable(table []models.Domino) models.TableMap {
	tableMap := make(models.TableMap, len(table))
	for _, v := range table {
		if _, ok := tableMap[v.L]; !ok {
			tableMap[v.L] = make(models.TableBone, len(table))
		}

		if _, ok := tableMap[v.R]; !ok {
			tableMap[v.R] = make(models.TableBone, len(table))
		}

		tableMap[v.L][v.R] = true
		tableMap[v.R][v.L] = true
	}

	return tableMap
}
