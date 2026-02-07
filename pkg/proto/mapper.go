package proto

import (
	"time"

	"github.com/trytobebee/snake_go/pkg/game"
)

func ToProtoPoint(p game.Point) *Point {
	return &Point{X: int32(p.X), Y: int32(p.Y)}
}

func FromProtoPoint(p *Point) game.Point {
	if p == nil {
		return game.Point{}
	}
	return game.Point{X: int(p.X), Y: int(p.Y)}
}

func ToProtoGameState(gs game.GameState) *GameStateSnapshot {
	snake := make([]*Point, len(gs.Snake))
	for i, p := range gs.Snake {
		snake[i] = ToProtoPoint(p)
	}

	foods := make([]*FoodInfo, len(gs.Foods))
	for i, f := range gs.Foods {
		foods[i] = &FoodInfo{
			Pos:              ToProtoPoint(f.Pos),
			FoodType:         int32(f.FoodType),
			RemainingSeconds: int32(f.RemainingSeconds),
		}
	}

	obstacles := make([]*Obstacle, len(gs.Obstacles))
	for i, o := range gs.Obstacles {
		pts := make([]*Point, len(o.Points))
		for j, p := range o.Points {
			pts[j] = ToProtoPoint(p)
		}
		obstacles[i] = &Obstacle{
			Points:   pts,
			Duration: o.Duration,
		}
	}

	fireballs := make([]*Fireball, len(gs.Fireballs))
	for i, f := range gs.Fireballs {
		fireballs[i] = &Fireball{
			Pos:   ToProtoPoint(f.Pos),
			Dir:   ToProtoPoint(f.Dir),
			Owner: f.Owner,
		}
	}

	hitPoints := make([]*Point, len(gs.HitPoints))
	for i, p := range gs.HitPoints {
		hitPoints[i] = ToProtoPoint(p)
	}

	aiSnake := make([]*Point, len(gs.AISnake))
	for i, p := range gs.AISnake {
		aiSnake[i] = ToProtoPoint(p)
	}

	scoreEvents := make([]*ScoreEvent, len(gs.ScoreEvents))
	for i, se := range gs.ScoreEvents {
		scoreEvents[i] = &ScoreEvent{
			Pos:    ToProtoPoint(se.Pos),
			Amount: int32(se.Amount),
			Label:  se.Label,
		}
	}

	var crashPoint *Point
	if gs.CrashPoint != nil {
		crashPoint = ToProtoPoint(*gs.CrashPoint)
	}

	props := make([]*Prop, len(gs.Props))
	for i, p := range gs.Props {
		props[i] = &Prop{
			Pos:  ToProtoPoint(p.Pos),
			Type: int32(p.Type),
		}
	}

	p1Effects := make([]*ActiveEffect, len(gs.P1Effects))
	for i, e := range gs.P1Effects {
		p1Effects[i] = &ActiveEffect{
			Type:     string(e.Type),
			Duration: e.Duration,
		}
	}

	p2Effects := make([]*ActiveEffect, len(gs.P2Effects))
	for i, e := range gs.P2Effects {
		p2Effects[i] = &ActiveEffect{
			Type:     string(e.Type),
			Duration: e.Duration,
		}
	}

	return &GameStateSnapshot{
		Snake:         snake,
		Foods:         foods,
		Score:         int32(gs.Score),
		FoodEaten:     int32(gs.FoodEaten),
		EatingSpeed:   gs.EatingSpeed,
		Started:       gs.Started,
		GameOver:      gs.GameOver,
		Paused:        gs.Paused,
		Boosting:      gs.Boosting,
		AutoPlay:      gs.AutoPlay,
		Difficulty:    gs.Difficulty,
		Message:       gs.Message,
		MessageType:   gs.MessageType,
		CrashPoint:    crashPoint,
		Obstacles:     obstacles,
		Fireballs:     fireballs,
		HitPoints:     hitPoints,
		AiSnake:       aiSnake,
		AiScore:       int32(gs.AIScore),
		TimeRemaining: int32(gs.TimeRemaining),
		Winner:        gs.Winner,
		AiStunned:     gs.AIStunned,
		PlayerStunned: gs.PlayerStunned,
		Mode:          gs.Mode,
		ScoreEvents:   scoreEvents,
		Berserker:     gs.Berserker,
		IsPVP:         gs.IsPVP,
		P1Name:        gs.P1Name,
		P2Name:        gs.P2Name,
		Props:         props,
		P1Effects:     p1Effects,
		P2Effects:     p2Effects,
	}
}

func ToProtoConfig(c *game.GameConfig) *GameConfig {
	if c == nil {
		return nil
	}
	return &GameConfig{
		Width:            int32(c.Width),
		Height:           int32(c.Height),
		GameDuration:     int32(c.GameDuration),
		FireballCooldown: int32(c.FireballCooldown),
	}
}

func ToProtoLeaderboard(entries []game.LeaderboardEntry) []*LeaderboardEntry {
	res := make([]*LeaderboardEntry, len(entries))
	for i, e := range entries {
		res[i] = &LeaderboardEntry{
			Name:       e.Name,
			Score:      int32(e.Score),
			Date:       e.Date.Format(time.RFC3339),
			Difficulty: e.Difficulty,
			Mode:       e.Mode,
		}
	}
	return res
}

func ToProtoWinRates(entries []game.WinRateEntry) []*WinRateEntry {
	res := make([]*WinRateEntry, len(entries))
	for i, e := range entries {
		res[i] = &WinRateEntry{
			Name:       e.Name,
			WinRate:    e.WinRate,
			TotalWins:  int32(e.TotalWins),
			TotalGames: int32(e.TotalGames),
		}
	}
	return res
}

func ToProtoUser(u *game.User) *User {
	if u == nil {
		return nil
	}
	return &User{
		Username:   u.Username,
		BestScore:  int32(u.BestScore),
		TotalGames: int32(u.TotalGames),
		TotalWins:  int32(u.TotalWins),
		CreatedAt:  u.CreatedAt.Format(time.RFC3339),
	}
}

func ToProtoServerMessage(typeStr string, config *game.GameConfig, state *game.GameState, leaderboard []game.LeaderboardEntry, winRates []game.WinRateEntry, user *game.User, errStr string, successStr string, sessionCount int) *ServerMessage {
	var protoState *GameStateSnapshot
	if state != nil {
		protoState = ToProtoGameState(*state)
	}

	return &ServerMessage{
		Type:         typeStr,
		Config:       ToProtoConfig(config),
		State:        protoState,
		Leaderboard:  ToProtoLeaderboard(leaderboard),
		WinRates:     ToProtoWinRates(winRates),
		User:         ToProtoUser(user),
		Error:        errStr,
		Success:      successStr,
		SessionCount: int32(sessionCount),
	}
}
