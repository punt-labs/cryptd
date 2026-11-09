package protocol

import (
	"encoding/json"
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
)

func TestRPCError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *RPCError
		want string
	}{
		{
			name: "nil receiver",
			err:  nil,
			want: "<nil>",
		},
		{
			name: "code and message",
			err:  &RPCError{Code: CodeInvalidParams, Message: "direction is required"},
			want: "rpc error -32602: direction is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRPCError_SatisfiesError verifies *RPCError is assignable to error.
func TestRPCError_SatisfiesError(t *testing.T) {
	var err error = &RPCError{Code: CodeInternalError, Message: "boom"}
	if err.Error() == "" {
		t.Fatal("expected non-empty error string")
	}
}

// Wire-contract tests. These pin the omit-vs-null-vs-present choice of every
// conditional key in a tool result. The dispatcher sets a pointer field only
// in specific states; the JSON a client decodes must reflect exactly that.
// A test here fails the moment someone adds or drops an omitempty, which is a
// wire-breaking change for the crypt client.

// keyState classifies how a JSON key appears after marshaling.
type keyState int

const (
	keyAbsent  keyState = iota // key not present at all
	keyNull                    // key present with value null
	keyPresent                 // key present with a non-null value
)

// marshalKeys marshals v and reports the state of each requested key.
func marshalKeys(t *testing.T, v any, keys ...string) map[string]keyState {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to raw: %v (json was %s)", err, data)
	}
	got := make(map[string]keyState, len(keys))
	for _, k := range keys {
		msg, ok := raw[k]
		switch {
		case !ok:
			got[k] = keyAbsent
		case string(msg) == "null":
			got[k] = keyNull
		default:
			got[k] = keyPresent
		}
	}
	return got
}

func assertKey(t *testing.T, got map[string]keyState, key string, want keyState) {
	t.Helper()
	names := map[keyState]string{keyAbsent: "absent", keyNull: "null", keyPresent: "present"}
	if got[key] != want {
		t.Errorf("key %q = %s, want %s", key, names[got[key]], names[want])
	}
}

func TestWireContract_AttackResult_NilOutcome(t *testing.T) {
	// No combat outcome: the embedded *CombatOutcome is nil, so its keys and
	// the enemy_turns key are all absent.
	r := AttackResult{Target: "goblin", Damage: 3, Killed: false}
	got := marshalKeys(t, r, "combat_over", "level_up", "enemy_turns", "xp_awarded")
	assertKey(t, got, "combat_over", keyAbsent)
	assertKey(t, got, "level_up", keyAbsent)
	assertKey(t, got, "enemy_turns", keyAbsent)
	assertKey(t, got, "xp_awarded", keyAbsent)
}

func TestWireContract_AttackResult_OutcomeNilLevelUp(t *testing.T) {
	// Combat ended with no level gained: combat_over present, level_up null.
	r := AttackResult{
		Target:        "goblin",
		Killed:        true,
		CombatOutcome: &CombatOutcome{CombatOver: true, LevelUp: nil},
	}
	got := marshalKeys(t, r, "combat_over", "level_up")
	assertKey(t, got, "combat_over", keyPresent)
	assertKey(t, got, "level_up", keyNull)
}

func TestWireContract_AttackResult_OutcomeWithLevelUp(t *testing.T) {
	r := AttackResult{
		Target: "goblin",
		Killed: true,
		CombatOutcome: &CombatOutcome{
			CombatOver: true,
			LevelUp:    &LevelUpSummary{NewLevel: 2, HPGain: 5, MPGain: 1},
		},
	}
	got := marshalKeys(t, r, "combat_over", "level_up")
	assertKey(t, got, "combat_over", keyPresent)
	assertKey(t, got, "level_up", keyPresent)
}

func TestWireContract_MoveResult_NoCombat(t *testing.T) {
	// A move into an empty room: no combat, no enemy phase, no warning. All
	// three conditional keys are absent.
	r := MoveResult{Room: "hall", Name: "Hall"}
	got := marshalKeys(t, r, "combat", "enemy_turns", "warning")
	assertKey(t, got, "combat", keyAbsent)
	assertKey(t, got, "enemy_turns", keyAbsent)
	assertKey(t, got, "warning", keyAbsent)
}

func TestWireContract_MoveResult_CombatStarted(t *testing.T) {
	// A move that triggers combat sets the *CombatSummary: the combat key is a
	// present object. A warning string also emits once non-empty.
	r := MoveResult{
		Room:    "lair",
		Combat:  &CombatSummary{Enemies: []CombatEnemy{{ID: "g1", Name: "Goblin"}}, HeroTurn: true},
		Warning: "a goblin blocks your path",
	}
	got := marshalKeys(t, r, "combat", "warning")
	assertKey(t, got, "combat", keyPresent)
	assertKey(t, got, "warning", keyPresent)
}

func TestWireContract_LookResult_NoCombat(t *testing.T) {
	// Looking outside combat leaves the *CombatView nil: combat key absent.
	r := LookResult{Room: "hall", Name: "Hall"}
	got := marshalKeys(t, r, "combat")
	assertKey(t, got, "combat", keyAbsent)
}

func TestWireContract_LookResult_InCombat(t *testing.T) {
	// Looking during combat sets the *CombatView: combat key is a present object.
	r := LookResult{
		Room:   "lair",
		Combat: &CombatView{Enemies: []CombatEnemy{{ID: "g1", Name: "Goblin"}}, Round: 1, HeroTurn: true},
	}
	got := marshalKeys(t, r, "combat")
	assertKey(t, got, "combat", keyPresent)
}

func TestWireContract_AttackResult_XPAwardedPresent(t *testing.T) {
	// A kill that awards XP sets *XPAwarded; a real zero award still emits
	// "xp_awarded":0 rather than being dropped.
	zero := 0
	r := AttackResult{Target: "goblin", Killed: true, XPAwarded: &zero}
	got := marshalKeys(t, r, "xp_awarded")
	assertKey(t, got, "xp_awarded", keyPresent)

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(decoded["xp_awarded"]) != "0" {
		t.Errorf("xp_awarded = %s, want 0", decoded["xp_awarded"])
	}
}

func TestWireContract_EnemyTurn_HeroDead(t *testing.T) {
	// A killing blow sets HeroDead true, so the omitempty key emits "hero_dead":true.
	dmg := 5
	hp := 0
	turn := EnemyTurn{Enemy: "goblin", Action: "attack", Damage: &dmg, HeroHP: &hp, HeroDead: true}
	got := marshalKeys(t, turn, "hero_dead")
	assertKey(t, got, "hero_dead", keyPresent)

	data, err := json.Marshal(turn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(decoded["hero_dead"]) != "true" {
		t.Errorf("hero_dead = %s, want true", decoded["hero_dead"])
	}
}

func TestWireContract_EnemyTurns_NilPointer(t *testing.T) {
	// No enemy phase: the *[]EnemyTurn is nil, key absent.
	r := DefendResult{Defending: true}
	got := marshalKeys(t, r, "enemy_turns")
	assertKey(t, got, "enemy_turns", keyAbsent)
}

func TestWireContract_EnemyTurns_EmptySlicePointer(t *testing.T) {
	// A non-nil pointer to a nil slice must emit null, not be dropped: the
	// dispatcher distinguishes "enemy phase ran, no actions" from "no phase".
	var turns []EnemyTurn
	r := DefendResult{Defending: true, EnemyTurns: &turns}
	got := marshalKeys(t, r, "enemy_turns")
	assertKey(t, got, "enemy_turns", keyNull)
}

func TestWireContract_EnemyTurn_ZeroDamage(t *testing.T) {
	// Damage is *int so a real zero (hit for 0) emits "damage":0, not absent.
	zero := 0
	hp := 10
	turn := EnemyTurn{Enemy: "goblin", Action: "attack", Damage: &zero, HeroHP: &hp}
	got := marshalKeys(t, turn, "damage", "hero_hp", "hero_dead")
	assertKey(t, got, "damage", keyPresent)
	assertKey(t, got, "hero_hp", keyPresent)
	// hero_dead is a bool with omitempty: false is dropped.
	assertKey(t, got, "hero_dead", keyAbsent)

	data, err := json.Marshal(turn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(decoded["damage"]) != "0" {
		t.Errorf("damage = %s, want 0", decoded["damage"])
	}
}

func TestWireContract_EnemyTurn_NonAttack(t *testing.T) {
	// A non-attack action leaves Damage and HeroHP nil, so both keys are absent.
	turn := EnemyTurn{Enemy: "goblin", Action: "skip"}
	got := marshalKeys(t, turn, "damage", "hero_hp")
	assertKey(t, got, "damage", keyAbsent)
	assertKey(t, got, "hero_hp", keyAbsent)
}

func TestWireContract_CastSpellResult_TargetOmitted(t *testing.T) {
	// A non-damage spell leaves Target nil: target key absent.
	r := CastSpellResult{Spell: "heal", Effect: "heal", Power: 5}
	got := marshalKeys(t, r, "target", "combat_over", "enemy_turns")
	assertKey(t, got, "target", keyAbsent)
	assertKey(t, got, "combat_over", keyAbsent)
	assertKey(t, got, "enemy_turns", keyAbsent)
}

func TestWireContract_CastSpellResult_TargetPresent(t *testing.T) {
	// A damage spell sets Target, even when it is the empty string.
	target := ""
	r := CastSpellResult{Spell: "firebolt", Effect: "damage", Power: 8, Target: &target}
	got := marshalKeys(t, r, "target")
	assertKey(t, got, "target", keyPresent)
}

func TestWireContract_CurrentRoomContext_DetailOmitted(t *testing.T) {
	// Id-only fallback: the embedded *RoomContextDetail is nil, so its keys
	// (name, description_seed, exits, items, enemies) are all absent.
	r := CurrentRoomContext{ID: "void"}
	got := marshalKeys(t, r, "id", "name", "description_seed", "exits", "items", "enemies")
	assertKey(t, got, "id", keyPresent)
	assertKey(t, got, "name", keyAbsent)
	assertKey(t, got, "description_seed", keyAbsent)
	assertKey(t, got, "exits", keyAbsent)
	assertKey(t, got, "items", keyAbsent)
	assertKey(t, got, "enemies", keyAbsent)
}

func TestWireContract_CurrentRoomContext_DetailPresent(t *testing.T) {
	r := CurrentRoomContext{
		ID: "entrance",
		RoomContextDetail: &RoomContextDetail{
			Name:            "Entrance Hall",
			DescriptionSeed: "cold stone",
			Exits:           []string{"north"},
			Items:           []string{},
			Enemies:         []string{},
		},
	}
	got := marshalKeys(t, r, "id", "name", "description_seed", "exits", "items", "enemies")
	for _, k := range []string{"id", "name", "description_seed", "exits", "items", "enemies"} {
		assertKey(t, got, k, keyPresent)
	}
}

func TestWireContract_AdjacentRoom_DetailOmitted(t *testing.T) {
	// A neighbour not found in the scenario has no detail: name and
	// description_seed absent, the three base keys present.
	r := AdjacentRoom{ID: "unknown", Direction: "north", Visited: false}
	got := marshalKeys(t, r, "id", "direction", "visited", "name", "description_seed")
	assertKey(t, got, "id", keyPresent)
	assertKey(t, got, "direction", keyPresent)
	assertKey(t, got, "visited", keyPresent)
	assertKey(t, got, "name", keyAbsent)
	assertKey(t, got, "description_seed", keyAbsent)
}

func TestWireContract_AdjacentRoom_DetailPresent(t *testing.T) {
	r := AdjacentRoom{
		ID:                 "hall",
		Direction:          "north",
		Visited:            true,
		AdjacentRoomDetail: &AdjacentRoomDetail{Name: "Hall", DescriptionSeed: "dim"},
	}
	got := marshalKeys(t, r, "name", "description_seed")
	assertKey(t, got, "name", keyPresent)
	assertKey(t, got, "description_seed", keyPresent)
}

func TestWireContract_HeroContext_AlwaysEmitsInventoryAndEquipped(t *testing.T) {
	// The context tool emits inventory and equipped even when empty: inventory
	// is a nil slice (null), equipped is a zero struct (present object).
	r := HeroContext{HeroSummary: HeroSummary{Name: "Tester"}}
	got := marshalKeys(t, r, "inventory", "equipped", "name")
	assertKey(t, got, "name", keyPresent)
	assertKey(t, got, "inventory", keyNull)
	assertKey(t, got, "equipped", keyPresent)
}

func TestWireContract_HeroContext_PopulatedInventory(t *testing.T) {
	r := HeroContext{
		HeroSummary: HeroSummary{Name: "Tester"},
		Inventory:   []string{"short_sword"},
		Equipped:    model.Equipment{Weapon: "short_sword"},
	}
	got := marshalKeys(t, r, "inventory", "equipped")
	assertKey(t, got, "inventory", keyPresent)
	assertKey(t, got, "equipped", keyPresent)
}
