package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/henok321/knobel-manager-service/gen/api"
	"github.com/henok321/knobel-manager-service/pkg/game"
	"github.com/henok321/knobel-manager-service/pkg/table"
)

type TablesHandler struct {
	gamesService  *game.GamesService
	tablesService *table.TablesService
}

func NewTablesHandler(gamesService *game.GamesService, tablesService *table.TablesService) *TablesHandler {
	return &TablesHandler{gamesService: gamesService, tablesService: tablesService}
}

func (t *TablesHandler) GetGameTables(writer http.ResponseWriter, request *http.Request, gameID int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	gameByID, err := t.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		respondError(ctx, writer, err)
		return
	}

	apiTables := make([]api.Table, 0)

	for _, round := range gameByID.Rounds {
		for _, currentTable := range round.Tables {
			apiTables = append(apiTables, entityTableToAPITable(*currentTable))
		}
	}

	writeJSON(ctx, writer, http.StatusOK, api.TablesResponse{Tables: apiTables})
}

func (t *TablesHandler) GetTables(writer http.ResponseWriter, request *http.Request, gameID, roundNumber int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	gameByID, err := t.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		respondError(ctx, writer, err)
		return
	}

	for _, round := range gameByID.Rounds {
		if round.RoundNumber == roundNumber {
			apiTables := make([]api.Table, len(round.Tables))
			for i, t := range round.Tables {
				apiTables[i] = entityTableToAPITable(*t)
			}

			writeJSON(ctx, writer, http.StatusOK, api.TablesResponse{Tables: apiTables})

			return
		}
	}

	JSONError(ctx, writer, http.StatusNotFound, "Round not found")
}

func (t *TablesHandler) GetTable(writer http.ResponseWriter, request *http.Request, gameID, roundNumber, tableNumber int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	gameByID, err := t.gamesService.FindByID(ctx, gameID, sub)
	if err != nil {
		respondError(ctx, writer, err)
		return
	}

	for _, round := range gameByID.Rounds {
		if round.RoundNumber == roundNumber {
			for _, currentTable := range round.Tables {
				if currentTable.TableNumber == tableNumber {
					writeJSON(ctx, writer, http.StatusOK, api.TableResponse{Table: entityTableToAPITable(*currentTable)})

					return
				}
			}
		}
	}

	JSONError(ctx, writer, http.StatusNotFound, "Round or table not found")
}

func (t *TablesHandler) UpdateScores(writer http.ResponseWriter, request *http.Request, gameID, roundNumber, tableNumber int) {
	ctx := request.Context()

	sub, ok := userSub(writer, request)
	if !ok {
		return
	}

	scoresRequest := api.ScoresRequest{}

	if err := json.NewDecoder(request.Body).Decode(&scoresRequest); err != nil {
		JSONError(ctx, writer, http.StatusBadRequest, err.Error())
		return
	}

	if len(scoresRequest.Scores) == 0 {
		JSONError(ctx, writer, http.StatusBadRequest, "Invalid request body")
		return
	}

	updatedTable, err := t.tablesService.UpdateScore(ctx, gameID, roundNumber, tableNumber, sub, scoresRequest)
	if err != nil {
		respondError(ctx, writer, err)
		return
	}

	writeJSON(ctx, writer, http.StatusOK, api.TableResponse{Table: entityTableToAPITable(updatedTable)})
}
