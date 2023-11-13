package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/josecleiton/domino/app/game"
	"github.com/josecleiton/domino/app/models"
)

type gameStateRequest struct {
	Player int                `json:"jogador"`
	Hand   []string           `json:"mao"`
	Table  []string           `json:"mesa"`
	Plays  []playStateRequest `json:"jogadas"`
}

type playStateRequest struct {
	Player    int    `json:"jogador"`
	Bone      string `json:"pedra"`
	Direction string `json:"lado"`
}

type playStateResponse struct {
	Player    int     `json:"jogador"`
	Bone      *string `json:"pedra"`
	Direction *string `json:"lado"`
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

	play := game.Play(domino)

	resp := dominoPlayToResponse(*play)
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Printf("Error happened in JSON marshal. Err: %s\n", err)
	}

	w.Write(jsonResp)
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
	table := make(map[int]map[int]bool, models.DominoUniqueBones)
	plays := make([]models.DominoPlay, 0, len(request.Plays))

	for _, bone := range request.Hand {
		domino, err := models.DominoFromString(bone)
		if err != nil {
			return nil, err
		}

		hand = append(hand, *domino)
	}

	for _, bone := range request.Table {
		domino, err := models.DominoFromString(bone)
		if err != nil {
			return nil, err
		}

		if _, ok := table[domino.X]; !ok {
			table[domino.X] = make(map[int]bool, models.DominoUniqueBones)
		}
		if _, ok := table[domino.Y]; !ok {
			table[domino.Y] = make(map[int]bool, models.DominoUniqueBones)
		}

		table[domino.X][domino.Y] = true
		table[domino.Y][domino.X] = true
	}

	for _, play := range request.Plays {
		domino, err := models.DominoFromString(play.Bone)
		if err != nil {
			return nil, err
		}

		plays = append(plays, models.DominoPlay{
			PlayerPosition: play.Player,
			Bone: models.DominoInTable{
				Domino:   *domino,
				Reversed: strings.HasPrefix(strings.ToLower(play.Direction), "d"),
			},
		})
	}

	return &models.DominoGameState{
		PlayerPosition: request.Player,
		Hand:           hand,
		Table:          table,
		Plays:          plays,
	}, nil
}

func dominoPlayToResponse(dominoPlay models.DominoPlayWithPass) *playStateResponse {
	if dominoPlay.Pass() {
		return &playStateResponse{Player: dominoPlay.PlayerPosition}
	}

	direction := "esquerda"

	if dominoPlay.Bone.Reversed {
		direction = "direita"
	}

	bone := dominoPlay.Bone.Domino.String()
	return &playStateResponse{
		Player:    dominoPlay.PlayerPosition,
		Bone:      &bone,
		Direction: &direction,
	}
}
