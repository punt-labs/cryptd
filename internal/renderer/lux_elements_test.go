package renderer

import (
	"testing"

	"github.com/punt-labs/cryptd/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSceneToElements_RoomHeader(t *testing.T) {
	scene := LuxScene{Room: "entrance"}
	elements := SceneToElements(scene)

	require.NotEmpty(t, elements)
	header := elements[0]
	assert.Equal(t, "text", header["kind"])
	assert.Equal(t, "room_header", header["id"])
	assert.Equal(t, "entrance", header["content"])
	assert.Equal(t, "heading", header["style"])
}

func TestSceneToElements_PartyHP(t *testing.T) {
	scene := LuxScene{
		Room: "entrance",
		Party: []LuxHero{
			{Name: "Hero", HP: 75, MaxHP: 100},
		},
	}
	elements := SceneToElements(scene)

	// Find the party group.
	var partyGroup map[string]any
	for _, el := range elements {
		if el["id"] == "party" {
			partyGroup = el
			break
		}
	}
	require.NotNil(t, partyGroup, "party group element must exist")
	assert.Equal(t, "group", partyGroup["kind"])
	assert.Equal(t, "columns", partyGroup["layout"])

	children := partyGroup["children"].([]map[string]any)
	require.Len(t, children, 1)
	assert.Equal(t, "progress", children[0]["kind"])
	assert.Equal(t, "hero_0_hp", children[0]["id"])
	assert.Equal(t, 0.75, children[0]["fraction"])
	assert.Equal(t, "Hero HP 75/100", children[0]["label"])
}

func TestSceneToElements_ActionButtons(t *testing.T) {
	scene := LuxScene{
		Room:    "entrance",
		Actions: []string{"south", "look", "inventory"},
	}
	elements := SceneToElements(scene)

	// Action buttons are inside a group with id "actions".
	var actionsGroup map[string]any
	for _, el := range elements {
		if el["id"] == "actions" {
			actionsGroup = el
			break
		}
	}
	require.NotNil(t, actionsGroup, "actions group not found")
	children, ok := actionsGroup["children"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, children, 3)
	assert.Equal(t, "act_south", children[0]["id"])
	assert.Equal(t, "south", children[0]["label"])
	assert.Equal(t, "act_look", children[1]["id"])
	assert.Equal(t, "act_inventory", children[2]["id"])
}

func TestSceneToElements_Narration(t *testing.T) {
	scene := LuxScene{
		Room:      "entrance",
		Narration: "You stand at the entrance.",
	}
	elements := SceneToElements(scene)

	var narration map[string]any
	for _, el := range elements {
		if el["id"] == "narration" {
			narration = el
			break
		}
	}
	require.NotNil(t, narration)
	assert.Equal(t, "markdown", narration["kind"])
	assert.Equal(t, "You stand at the entrance.", narration["content"])
}

func TestSceneToElements_CombatEnemies(t *testing.T) {
	scene := LuxScene{
		Room:     "goblin_lair",
		InCombat: true,
		Enemies: []LuxEnemy{
			{Name: "Goblin", HP: 4, MaxHP: 8},
		},
	}
	elements := SceneToElements(scene)

	var enemyBar map[string]any
	for _, el := range elements {
		if el["id"] == "enemy_goblin_0_hp" {
			enemyBar = el
			break
		}
	}
	require.NotNil(t, enemyBar)
	assert.Equal(t, "progress", enemyBar["kind"])
	assert.Equal(t, 0.5, enemyBar["fraction"])
	assert.Equal(t, "Goblin HP 4/8", enemyBar["label"])
}

func TestUpdateToPatches_Narration(t *testing.T) {
	update := LuxUpdate{
		Type:    "narration",
		Content: "You look around.",
	}
	patches := UpdateToPatches(update)

	require.NotEmpty(t, patches)
	assert.Equal(t, "narration", patches[0]["id"])
	set := patches[0]["set"].(map[string]any)
	assert.Equal(t, "You look around.", set["content"])
}

func TestUpdateToPatches_HeroHP(t *testing.T) {
	hero := LuxHero{Name: "Hero", HP: 80, MaxHP: 100}
	update := LuxUpdate{
		Type:    "narration",
		Content: "ouch",
		Hero:    &hero,
	}
	patches := UpdateToPatches(update)

	var heroPatch map[string]any
	for _, p := range patches {
		if p["id"] == "hero_0_hp" {
			heroPatch = p
			break
		}
	}
	require.NotNil(t, heroPatch)
	set := heroPatch["set"].(map[string]any)
	assert.Equal(t, 0.8, set["fraction"])
	assert.Equal(t, "Hero HP 80/100", set["label"])
}

func TestTranslateLuxEvent_ButtonClick(t *testing.T) {
	event := map[string]any{
		"element_id": "act_south",
		"action":     "clicked",
	}
	input, ok := TranslateLuxEvent(event)
	require.True(t, ok)
	assert.Equal(t, model.InputEvent{Type: "input", Payload: "south"}, input)
}

func TestTranslateLuxEvent_UnknownID(t *testing.T) {
	tests := []struct {
		name  string
		event map[string]any
	}{
		{"no act_ prefix", map[string]any{"element_id": "room_header", "action": "clicked"}},
		{"not clicked", map[string]any{"element_id": "act_south", "action": "hovered"}},
		{"empty command", map[string]any{"element_id": "act_", "action": "clicked"}},
		{"empty map", map[string]any{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := TranslateLuxEvent(tt.event)
			assert.False(t, ok)
		})
	}
}
