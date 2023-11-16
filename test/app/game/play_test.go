package game

import (
	"fmt"
	"testing"

	"github.com/josecleiton/domino/app/game"
	"github.com/josecleiton/domino/app/models"
)

func firstPlay() models.DominoGameState {
	return models.DominoGameState{
		PlayerPosition: 1,
		Hand: []models.Domino{
			{X: 0, Y: 0},
			{X: 0, Y: 3},
			{X: 1, Y: 2},
			{X: 1, Y: 3},
			{X: 1, Y: 6},
			{X: 3, Y: 6},
			{X: 5, Y: 5},
		},
		TableMap: models.TableMap{},
		Table:    []models.Domino{},
		Plays:    []models.DominoPlay{},
		Edges:    models.Edges{},
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
					X: 4,
					Y: 5,
				},
			},
		},
		{
			PlayerPosition: stPlay.PlayerPosition + 2,
			Bone: models.DominoInTable{
				Edge: models.RightEdge,
				Domino: models.Domino{
					X: 5,
					Y: 3,
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
		if _, ok := tableMap[play.Bone.X]; !ok {
			tableMap[play.Bone.X] = make(models.TableBone, len(plays))
		}

		if _, ok := tableMap[play.Bone.Y]; !ok {
			tableMap[play.Bone.Y] = make(models.TableBone, len(plays))
		}

		tableMap[play.Bone.X][play.Bone.Y] = true
		tableMap[play.Bone.Y][play.Bone.X] = true
	}

	newHand := make([]models.Domino, 0, len(stGameState.Hand)-1)
	for _, bone := range stGameState.Hand {
		if bone != stPlay.Bone.Domino {
			newHand = append(newHand, bone)
		}
	}

	edges := models.Edges{
		models.LeftEdge:  &plays[1].Bone.Domino,
		models.RightEdge: &plays[2].Bone.Domino,
	}

	return models.DominoGameState{
		PlayerPosition: stPlay.PlayerPosition,
		Edges:          edges,
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
	fmt.Println("edges:", ndGameState.Edges)
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
		fmt.Printf("Bone %v not found in hand %v\n", ndPlay.Bone.Domino, ndGameState.Hand)
		t.Fail()
	}

	fmt.Println("newPlay:", ndPlay)
}

func TestPassedPlay(t *testing.T) {
	tableSt := []models.Domino{
		{X: 0, Y: 2},
		{X: 2, Y: 5},
		{X: 5, Y: 5},
		{X: 5, Y: 6},
		{X: 6, Y: 6},
		{X: 6, Y: 0},
		{X: 0, Y: 3},
		{X: 3, Y: 3},
		{X: 3, Y: 6},
	}
	tableMapSt := tableMapFromTable(tableSt)
	gameStateSt := &models.DominoGameState{
		PlayerPosition: 2,
		Hand: []models.Domino{
			{X: 6, Y: 1},
			{X: 5, Y: 3},
			{X: 1, Y: 0},
			{X: 0, Y: 0},
			{X: 4, Y: 3},
		},
		Edges: models.Edges{
			models.LeftEdge: &models.Domino{
				X: 0,
				Y: 2,
			},
			models.RightEdge: &models.Domino{
				X: 3,
				Y: 6,
			},
		},
		Plays: []models.DominoPlay{
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 6, Y: 6},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 5, Y: 6},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 6, Y: 0},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 5, Y: 5},
				},
			},
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 2, Y: 5},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 0, Y: 3},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 0, Y: 2},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 3, Y: 3},
				},
			},
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 3, Y: 6},
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
		{X: 5, Y: 0},
		{X: 0, Y: 2},
		{X: 2, Y: 5},
		{X: 5, Y: 5},
		{X: 5, Y: 6},
		{X: 6, Y: 6},
		{X: 6, Y: 0},
		{X: 0, Y: 3},
		{X: 3, Y: 3},
		{X: 3, Y: 6},
		{X: 6, Y: 1},
		{X: 1, Y: 3},
	}
	tableMapNd := tableMapFromTable(tableNd)

	gameStateNd := &models.DominoGameState{
		PlayerPosition: 2,
		Hand: []models.Domino{
			{X: 5, Y: 3},
			{X: 1, Y: 0},
			{X: 0, Y: 0},
			{X: 4, Y: 3},
		},
		Edges: models.Edges{
			models.LeftEdge: &models.Domino{
				X: 5,
				Y: 0,
			},
			models.RightEdge: &models.Domino{
				X: 1,
				Y: 3,
			},
		},
		Plays: []models.DominoPlay{
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 6, Y: 6},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 5, Y: 6},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 6, Y: 0},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 5, Y: 5},
				},
			},
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 2, Y: 5},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 0, Y: 3},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 0, Y: 2},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 3, Y: 3},
				},
			},
			{
				PlayerPosition: 1,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 3, Y: 6},
				},
			},
			{
				PlayerPosition: 2,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 6, Y: 1},
				},
			},
			{
				PlayerPosition: 3,
				Bone: models.DominoInTable{
					Edge:   models.RightEdge,
					Domino: models.Domino{X: 1, Y: 3},
				},
			},
			{
				PlayerPosition: 4,
				Bone: models.DominoInTable{
					Edge:   models.LeftEdge,
					Domino: models.Domino{X: 5, Y: 0},
				},
			},
		},
		Table:    tableNd,
		TableMap: tableMapNd,
	}

	play = game.Play(gameStateNd)

	fmt.Println("play:", play)

	if (play.Bone.Domino != models.Domino{X: 3, Y: 5}) {
		t.Fatal("Wrong play")
	}
}

func tableMapFromTable(table []models.Domino) models.TableMap {
	tableMap := make(models.TableMap, len(table))
	for _, v := range table {
		if _, ok := tableMap[v.X]; !ok {
			tableMap[v.X] = make(models.TableBone, len(table))
		}

		if _, ok := tableMap[v.Y]; !ok {
			tableMap[v.Y] = make(models.TableBone, len(table))
		}

		tableMap[v.X][v.Y] = true
		tableMap[v.Y][v.X] = true
	}

	return tableMap
}
