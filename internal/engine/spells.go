package engine

import (
	"fmt"

	"github.com/punt-labs/cryptd/internal/dice"
	"github.com/punt-labs/cryptd/internal/model"
)

// CastResult holds the outcome of casting a spell.
type CastResult struct {
	SpellName string
	MPCost    int
	Effect    string // "damage" or "heal"
	Power     int    // rolled magnitude
	TargetID  string // enemy ID for damage spells
	HeroHP    int    // hero HP after heal
}

// NotCasterError is returned when the character's class cannot cast spells.
type NotCasterError struct {
	Class string
}

func (e *NotCasterError) Error() string {
	return fmt.Sprintf("a %s cannot cast that spell", e.Class)
}

// InsufficientMPError is returned when the character lacks enough MP.
type InsufficientMPError struct {
	Have int
	Need int
}

func (e *InsufficientMPError) Error() string {
	return fmt.Sprintf("not enough MP (have %d, need %d)", e.Have, e.Need)
}

// UnknownSpellError is returned when the spell ID is not in the scenario.
type UnknownSpellError struct {
	SpellID string
}

func (e *UnknownSpellError) Error() string {
	return fmt.Sprintf("unknown spell %q", e.SpellID)
}

// CastSpell executes a spell. Damage spells require an active combat target;
// heal spells work in or out of combat.
func (e *Engine) CastSpell(state *model.GameState, spellID, targetID string) (CastResult, error) {
	h := hero(state)

	// Look up spell definition.
	if e.s.Spells == nil {
		return CastResult{}, &UnknownSpellError{SpellID: spellID}
	}
	spell, ok := e.s.Spells[spellID]
	if !ok {
		return CastResult{}, &UnknownSpellError{SpellID: spellID}
	}

	// Class gate: check if the character's class is allowed.
	if !classCanCast(h.Class, spell.Classes) {
		return CastResult{}, &NotCasterError{Class: h.Class}
	}

	// MP check.
	if h.MP < spell.MP {
		return CastResult{}, &InsufficientMPError{Have: h.MP, Need: spell.MP}
	}

	// Roll power.
	d, err := dice.Parse(spell.Power)
	if err != nil {
		return CastResult{}, fmt.Errorf("invalid spell power %q: %w", spell.Power, err)
	}
	power := d.Roll()
	if power < 1 {
		power = 1
	}

	// Deduct MP.
	h.MP -= spell.MP

	switch spell.Effect {
	case "damage":
		return e.castDamage(state, spell.Name, spell.MP, power, targetID)
	case "heal":
		return e.castHeal(state, spell.Name, spell.MP, power)
	default:
		return CastResult{}, fmt.Errorf("unknown spell effect %q", spell.Effect)
	}
}

func (e *Engine) castDamage(state *model.GameState, spellName string, mpCost, power int, targetID string) (CastResult, error) {
	combat := &state.Dungeon.Combat
	if !combat.Active {
		return CastResult{}, &NotInCombatError{}
	}
	if !e.isHeroTurn(combat) {
		return CastResult{}, &NotHeroTurnError{}
	}

	if targetID == "" {
		targetID = e.FirstAliveEnemy(state)
	}
	target := e.findEnemy(combat, targetID)
	if target == nil || target.HP <= 0 {
		return CastResult{}, &InvalidTargetError{TargetID: targetID}
	}

	target.HP -= power
	if target.HP < 0 {
		target.HP = 0
	}

	result := CastResult{
		SpellName: spellName,
		MPCost:    mpCost,
		Effect:    "damage",
		Power:     power,
		TargetID:  targetID,
	}

	if target.HP <= 0 {
		h := hero(state)
		h.XP += target.MaxHP
	}

	if e.allEnemiesDead(combat) {
		e.endCombat(state)
	} else {
		combat.HeroDefending = false
		e.advanceTurn(combat)
	}

	return result, nil
}

func (e *Engine) castHeal(state *model.GameState, spellName string, mpCost, power int) (CastResult, error) {
	h := hero(state)
	h.HP += power
	if h.HP > h.MaxHP {
		h.HP = h.MaxHP
	}

	// If in combat, consume the hero's turn.
	combat := &state.Dungeon.Combat
	if combat.Active {
		if !e.isHeroTurn(combat) {
			return CastResult{}, &NotHeroTurnError{}
		}
		combat.HeroDefending = false
		e.advanceTurn(combat)
	}

	return CastResult{
		SpellName: spellName,
		MPCost:    mpCost,
		Effect:    "heal",
		Power:     power,
		HeroHP:    h.HP,
	}, nil
}

// classCanCast checks if the character's class is in the spell's allowed list.
func classCanCast(class string, allowed []string) bool {
	for _, c := range allowed {
		if c == class {
			return true
		}
	}
	return false
}
