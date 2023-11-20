package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/josecleiton/domino/app/game"
	"github.com/josecleiton/domino/app/models"
)

type gameStateRequest struct {
	Player int                `json:"jogador"`
	Hand   []string           `json:"mao"`
	Table  []string           `json:"mesa"`
	Plays  []playStateRequest `json:"jogadas"`
}

type externalDirection string

const (
	Left  externalDirection = "esquerda"
	Right externalDirection = "direita"
)

type playStateRequest struct {
	Player    int                `json:"jogador"`
	Bone      string             `json:"pedra"`
	Direction *externalDirection `json:"lado"`
}

type playStateResponse struct {
	Player    models.PlayerPosition `json:"jogador"`
	Bone      *string               `json:"pedra,omitempty"`
	Direction *externalDirection    `json:"lado,omitempty"`
}

func GameHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	var request gameStateRequest

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&request)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		log.Printf("Error happened in JSON marshal. Err: %s\n", err)

		return
	}

	domino, err := gameRequestToDomain(&request)
	if err != nil {
		const status = http.StatusBadRequest
		errorMap := map[string]interface{}{
			"error":  err.Error(),
			"status": http.StatusText(status),
			"code":   status,
		}

		w.WriteHeader(status)

		jsonResp, marshalErr := json.Marshal(errorMap)
		if marshalErr != nil {
			log.Printf("Error happened in JSON marshal. Err: %s\n", marshalErr)
			w.Write([]byte(err.Error()))
		} else {
			w.Write(jsonResp)
		}

		log.Printf("Error happened in play. Err: %s\n", err)
		return
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf(
			"[REQ] {PlaysCount: %d, Table: %v}\n",
			len(domino.Plays),
			domino.Table,
		)
	}()

	play := game.Play(domino)

	resp := dominoPlayToResponse(domino, play)

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Printf("Error happened in JSON marshal. Err: %s\n", err)
	}

	w.Write(jsonResp)

	wg.Wait()
	log.Printf("[RES] %v\n", play)
}

func gameRequestToDomain(request *gameStateRequest) (*models.DominoGameState, error) {
	if request.Player < models.DominoMinPlayer || request.Player > models.DominoMaxPlayer {
		return nil, fmt.Errorf(
			"player must be between %d and %d, not %d",
			models.DominoMinPlayer,
			models.DominoMaxPlayer,
			request.Player,
		)
	}

	hand := make([]models.Domino, 0, len(request.Hand))
	tableMap := make(models.TableMap, models.DominoUniqueBones)
	table := make([]models.Domino, 0, len(request.Table))
	possibleEdges := []models.Edge{models.LeftEdge, models.RightEdge}
	edges := make(models.Edges, len(possibleEdges))
	plays := make([]models.DominoPlay, len(request.Plays))

	for _, bone := range request.Hand {
		domino, err := models.DominoFromString(bone)
		if err != nil {
			return nil, err
		}

		hand = append(hand, *domino)
	}

	for _, play := range request.Plays {
		domino, err := models.DominoFromString(play.Bone)
		if err != nil {
			return nil, err
		}

		if _, ok := tableMap[domino.L]; !ok {
			tableMap[domino.L] = make(models.TableBone, models.DominoUniqueBones)
		}
		if _, ok := tableMap[domino.R]; !ok {
			tableMap[domino.R] = make(models.TableBone, models.DominoUniqueBones)
		}

		tableMap[domino.L][domino.R] = true
		tableMap[domino.R][domino.L] = true

		edge := models.LeftEdge
		if play.Direction != nil && *play.Direction == Right {
			edge = models.RightEdge
		}

		plays = append(plays, models.DominoPlay{
			PlayerPosition: models.PlayerPosition(play.Player),
			Bone: models.DominoInTable{
				Domino: *domino,
				Edge:   edge,
			},
		})

	}

	for _, v := range request.Table {
		domino, err := models.DominoFromString(v)
		if err != nil {
			return nil, err
		}

		table = append(table, *domino)
	}

	playsRequestLen := len(request.Plays)
	if playsRequestLen > 1 {
		edges[models.LeftEdge] = &plays[0].Bone.Domino
		edges[models.RightEdge] = &plays[len(plays)-1].Bone.Domino
	} else if playsRequestLen == 1 {
		leftEdge := plays[0].Bone.Domino
		rightEdge := leftEdge

		edges[models.LeftEdge] = &leftEdge
		edges[models.RightEdge] = &rightEdge
	}

	return &models.DominoGameState{
		PlayerPosition: models.PlayerPosition(request.Player),
		Hand:           hand,
		TableMap:       tableMap,
		Table:          table,
		Plays:          plays,
	}, nil
}

func dominoPlayToResponse(state *models.DominoGameState, dominoPlay models.DominoPlayWithPass) *playStateResponse {
	if dominoPlay.Pass() {
		return &playStateResponse{Player: dominoPlay.PlayerPosition}
	}

	direction := new(externalDirection)
	*direction = Left

	if len(state.TableMap) == 0 {
		direction = nil
	} else if dominoPlay.Bone.Edge == models.RightEdge {
		*direction = Right
	}

	domino := dominoPlay.Bone.Domino
	bone := fmt.Sprintf("%d-%d", domino.L, domino.R)
	return &playStateResponse{
		Player:    dominoPlay.PlayerPosition,
		Bone:      &bone,
		Direction: direction,
	}
}
