package server

import (
	"strconv"

	"github.com/Zamiell/isaac-racing-server/models"
)

const (
	NumRankedSoloRacesForAverage = 100
	ThirtyMinutesInMilliseconds  = 30 * 60 * 1000
)

func leaderboardUpdateRankedSolo(race *Race) {
	// Iterate over the map to get the solo racer
	var racer *Racer
	for _, v := range race.Racers {
		racer = v
	}

	// Get their ranked solo results for this season
	var raceResults []models.RaceResult
	if v, err := db.RaceParticipants.GetNRankedSoloRaceResults(racer.ID, NumRankedSoloRacesForAverage); err != nil {
		logger.Error("Database error while getting the ranked solo times:", err)
		return
	} else {
		raceResults = v
	}

	var numForfeits int
	lowestTime := int64(ThirtyMinutesInMilliseconds)
	var sumTimes int64
	for _, raceResult := range raceResults {
		if raceResult.Place > 0 {
			// They finished
			sumTimes += raceResult.RunTime

			if raceResult.RunTime < lowestTime {
				lowestTime = raceResult.RunTime
			}
		} else {
			// They quit
			numForfeits++
		}
	}

	var averageTime float64
	var forfeitPenalty float64
	if len(raceResults) == numForfeits {
		// If they forfeited every race, then we will have a divide by 0 later on, so arbitrarily
		// set it to 30 minutes. (1000 * 60 * 30)
		averageTime = 1800000
		forfeitPenalty = 1800000
	} else {
		averageTime = float64(sumTimes) / float64(len(raceResults)-numForfeits)
		forfeitPenalty = averageTime * float64(numForfeits) / float64(len(raceResults))
	}

	// Update their stats in the database.
	if err := db.Users.SetStatsRankedSolo(
		racer.ID,
		int(averageTime),
		numForfeits,
		int(forfeitPenalty),
		lowestTime,
		race.Ruleset.StartingBuildIndex,
	); err != nil {
		logger.Error("Database error while setting the ranked solo stats for \""+racer.Name+"\":", err)
		return
	}
}

func leaderboardRecalculateRankedSoloAll() {
	// This is equal to either the format in the database, or "ranked_solo" as an arbitrary string
	// ("ranked_solo" is not a real format, but it lets the child function know what specified rows
	// to query)
	// ("ranked_solo" refers to the prefix on the "users" table name in the database)
	format := "ranked_solo"

	if err := db.Users.ResetRankedSoloAll(); err != nil {
		logger.Error("Database error while resetting the unseeded solo stats:", err)
		return
	}

	var allRaces []models.RaceHistory
	if v, err := db.Races.GetAllRacesForLeaderboard(format); err != nil {
		logger.Error("Database error while getting all of the races:", err)
		return
	} else {
		allRaces = v
	}

	// Go through every race for this particular format in the database
	for _, modelsRace := range allRaces {
		// Convert the "RaceHistory" struct to a "Race" struct
		race := &Race{}
		race.ID = int(modelsRace.RaceID.Int64)

		newFormat := format
		if format == "ranked_solo" {
			newFormat = "unseeded"
		}
		race.Ruleset.Format = RaceFormat(newFormat)

		race.Racers = make(map[string]*Racer)
		for _, modelsRacer := range modelsRace.RaceParticipants {
			racer := &Racer{
				ID:    int(modelsRacer.ID.Int64),
				Name:  modelsRacer.RacerName.String,
				Place: int(modelsRacer.RacerPlace.Int64),
			}
			race.Racers[modelsRacer.RacerName.String] = racer
		}

		// Pretend like this race just finished
		leaderboardUpdateRankedSolo(race)
	}

	// Fix the "Date of Last Race" column
	if err := db.Users.SetLastRace(format); err != nil {
		logger.Error("Database error while setting the last race:", err)
		return
	}

	logger.Info("Successfully reset the leaderboard for " + format + ".")
}

func leaderboardRecalculateRankedSoloSpecificUser(userID int) {
	// This is equal to either the format in the database, or "ranked_solo" as an arbitrary string
	// ("ranked_solo" is not a real format, but it lets the child function know what specified rows
	// to query)
	// ("ranked_solo" refers to the prefix on the "users" table name in the database)
	format := "ranked_solo"

	if err := db.Users.ResetRankedSolo(userID); err != nil {
		logger.Error("Database error while resetting the unseeded solo stats:", err)
		return
	}

	var allRaces []models.RaceHistory
	if v, err := db.Races.GetRankedSoloRacesForUser(format, userID); err != nil {
		logger.Error("Database error while getting the ranked solo races for user "+strconv.Itoa(userID)+":", err)
		return
	} else {
		allRaces = v
	}

	for _, modelsRace := range allRaces {
		// Convert the "RaceHistory" struct to a "Race" struct
		race := &Race{}
		race.ID = int(modelsRace.RaceID.Int64)

		newFormat := format
		if format == "ranked_solo" {
			newFormat = "unseeded"
		}
		race.Ruleset.Format = RaceFormat(newFormat)

		race.Racers = make(map[string]*Racer)
		for _, modelsRacer := range modelsRace.RaceParticipants {
			racer := &Racer{
				ID:    int(modelsRacer.ID.Int64),
				Name:  modelsRacer.RacerName.String,
				Place: int(modelsRacer.RacerPlace.Int64),
			}
			race.Racers[modelsRacer.RacerName.String] = racer
		}

		// Pretend like this race just finished
		leaderboardUpdateRankedSolo(race)
	}

	// Fix the "Date of Last Race" column
	if err := db.Users.SetLastRace(format); err != nil {
		logger.Error("Database error while setting the last race:", err)
		return
	}

	logger.Info("Successfully reset the ranked solo leaderboard for user:", userID)
}
