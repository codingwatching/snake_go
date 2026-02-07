package game

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/trytobebee/snake_go/pkg/config"
)

// GetDuration returns the effect's duration
func (p *Prop) GetDuration() time.Duration {
	switch p.Type {
	case PropShield:
		return 12 * time.Second
	case PropTimeWarp:
		return 6 * time.Second
	case PropMagnet:
		return 10 * time.Second
	case PropRapidFire:
		return 12 * time.Second
	case PropScatterShot:
		return 15 * time.Second
	case PropTrimmer, PropChestBig, PropChestSmall:
		return 0 // Instant effect
	default:
		return 0
	}
}

// GetEmoji returns the emoji for the prop type
func (p *Prop) GetEmoji() string {
	switch p.Type {
	case PropShield:
		return "🛡️"
	case PropTimeWarp:
		return "🌀"
	case PropTrimmer:
		return "✂️"
	case PropMagnet:
		return "🧲"
	case PropChestBig:
		return "👑" // Big chest emoji
	case PropChestSmall:
		return "💰" // Small chest emoji
	case PropRapidFire:
		return "⚡" // Rapid fire emoji
	case PropScatterShot:
		return "🌟" // Scatter shot emoji
	default:
		return "🎁"
	}
}

// GetEffectType returns relevant effect type
func (p *Prop) GetEffectType() EffectType {
	switch p.Type {
	case PropShield:
		return EffectShield
	case PropTimeWarp:
		return EffectTimeWarp
	case PropMagnet:
		return EffectMagnet
	case PropRapidFire:
		return EffectRapidFire
	case PropScatterShot:
		return EffectScatterShot
	case PropTrimmer, PropChestBig, PropChestSmall:
		return EffectNone // No duration-based effect
	default:
		return EffectNone
	}
}

// IsExpired checks if the prop on board has expired
func (p *Prop) IsExpired(currentTotalPaused time.Duration) bool {
	pausedSinceSpawn := currentTotalPaused - p.PausedTimeAtSpawn
	elapsed := time.Since(p.SpawnTime) - pausedSinceSpawn
	return elapsed > config.PropDuration
}

// TrySpawnProp attempts to spawn a random prop
func (g *Game) TrySpawnProp() {
	if time.Since(g.LastPropSpawn) < config.PropSpawnInterval {
		return
	}

	if rand.Intn(100) >= config.PropSpawnChance {
		return
	}

	if len(g.Props) >= config.MaxPropsOnBoard {
		return
	}

	for attempts := 0; attempts < 50; attempts++ {
		pos := Point{
			X: rand.Intn(23) + 1,
			Y: rand.Intn(23) + 1,
		}

		if !g.isCellEmpty(pos) {
			continue
		}

		// Weighted selection:
		// Shield: 1, TimeWarp: 0.5, Trimmer: 0.5, Magnet: 0.5, RapidFire: 1, ScatterShot: 1, BigChest: 2, SmallChest: 4
		// Scaled by 2: Shield: 2, TimeWarp: 1, Trimmer: 1, Magnet: 1, RapidFire: 2, ScatterShot: 2, BigChest: 4, SmallChest: 8 (Total=21)
		randVal := rand.Intn(21)
		var t PropType
		switch {
		case randVal < 2:
			t = PropShield
		case randVal < 3:
			t = PropTimeWarp
		case randVal < 4:
			t = PropTrimmer
		case randVal < 5:
			t = PropMagnet
		case randVal < 7:
			t = PropRapidFire
		case randVal < 9:
			t = PropScatterShot
		case randVal < 13:
			t = PropChestBig
		default:
			t = PropChestSmall
		}

		newProp := Prop{
			Pos:               pos,
			Type:              t,
			SpawnTime:         time.Now(),
			PausedTimeAtSpawn: g.GetTotalPausedTime(),
		}
		g.Props = append(g.Props, newProp)
		g.LastPropSpawn = time.Now()
		g.SetMessage(fmt.Sprintf("%s A mysterious item appeared!", newProp.GetEmoji()))
		break
	}
}
