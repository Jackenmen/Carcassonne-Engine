package elements

import (
	"github.com/YetAnotherSpieskowcy/Carcassonne-Engine/pkg/game/position"
)

type MeepleWithPosition struct {
	Meeple
	Position position.Position
}

func NewMeepleWithPosition(
	meeple Meeple,
	pos position.Position,
) MeepleWithPosition {
	return MeepleWithPosition{
		Meeple:   meeple,
		Position: pos,
	}
}

type ScoreReport struct {
	// ReceivedPoints[playerID (uint8)] = player's received points
	ReceivedPoints map[ID]uint32
	// ReturnedMeeples[playerID (uint8)][meeple type (MeepleType)] = number of returned meeples
	// for reference, see also: player.meepleCounts
	ReturnedMeeples map[ID][]MeepleWithPosition
}

func NewScoreReport() ScoreReport {
	return ScoreReport{
		ReceivedPoints:  map[ID]uint32{},
		ReturnedMeeples: map[ID][]MeepleWithPosition{},
	}
}

func (report *ScoreReport) IsEmpty() bool {
	return len(report.ReceivedPoints) == 0 && len(report.ReturnedMeeples) == 0
}

// Adds the contents of otherReport to the contents of this score report
func (report *ScoreReport) Join(otherReport ScoreReport) {
	for playerID, score := range otherReport.ReceivedPoints {
		report.ReceivedPoints[playerID] += score
	}

	for playerID, meeples := range otherReport.ReturnedMeeples {
		_, keyExists := report.ReturnedMeeples[playerID]
		if !keyExists {
			report.ReturnedMeeples[playerID] = []MeepleWithPosition{}
		}

		// add meeples
		report.ReturnedMeeples[playerID] = append(report.ReturnedMeeples[playerID], meeples...)

	}
}

func (report *ScoreReport) MeepleInReport(testedMeeple MeepleWithPosition) bool {
	for _, meeplesWithPosition := range report.ReturnedMeeples { // for each player
		for _, meeple := range meeplesWithPosition {
			if meeple.Position == testedMeeple.Position {
				return true
			}
		}
	}
	return false
}

// Returns a list of IDs of players that have the most meeples in the given map
func GetPlayersWithMostMeeples(meeples map[ID][]uint8) []ID {
	var max uint8
	winningPlayers := []ID{}
	for playerID, numMeeples := range meeples {
		for meepleType, meepleCount := range numMeeples {
			if meepleCount > 0 && MeepleType(meepleType) != NoneMeeple {
				// TODO: add excluding meeples like builder, etc. when they are implemented
				if meepleCount > max {
					max = meepleCount
					winningPlayers = nil // remove all values that are in array since there is a player with more meeples
					winningPlayers = append(winningPlayers, playerID)
				} else if meepleCount == max {
					winningPlayers = append(winningPlayers, playerID)
				}
			}
		}
	}
	return winningPlayers
}

/*
Create score report by checking meeples control on the same Fully Connected Feature (like a whole city/road etc), ignoring not scoring meeples.
Returns a score report
*/
func CalculateScoreReportOnMeeples(score int, meeples []MeepleWithPosition) ScoreReport {
	var mostMeeples = uint8(0)
	var scoredPlayers = []uint8{}
	playerMeeples := make(map[uint8]uint8)
	// count meeples, and find max
	for _, meeple := range meeples {
		_, existKey := playerMeeples[uint8(meeple.PlayerID)]
		if !existKey {
			playerMeeples[uint8(meeple.PlayerID)] = 0
		}
		playerMeeples[uint8(meeple.PlayerID)]++
		if playerMeeples[uint8(meeple.PlayerID)] > mostMeeples {
			mostMeeples = playerMeeples[uint8(meeple.PlayerID)]
		}
	}

	// find players with max
	for playerID, count := range playerMeeples {
		if count == mostMeeples {
			scoredPlayers = append(scoredPlayers, playerID)
		}
	}

	// -------- create report -------------
	scoreReport := NewScoreReport()

	for _, playerID := range scoredPlayers {
		scoreReport.ReceivedPoints[ID(playerID)] = uint32(score)
	}

	for _, meeple := range meeples {
		_, ok := scoreReport.ReturnedMeeples[meeple.PlayerID]
		if !ok {
			scoreReport.ReturnedMeeples[meeple.PlayerID] = []MeepleWithPosition{}
		}
		scoreReport.ReturnedMeeples[meeple.PlayerID] = append(scoreReport.ReturnedMeeples[meeple.PlayerID], meeple)
	}

	return scoreReport
}
