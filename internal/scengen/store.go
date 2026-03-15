package scengen

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// Store persists graphs and scenario content in a SQLite database.
// Used only by crypt-admin for authoring workflows.
type Store struct {
	db *sql.DB
}

// OpenStore opens or creates a SQLite database at the given path.
func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Enable WAL mode and foreign keys.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec %s: %w", pragma, err)
		}
	}

	return &Store{db: db}, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateSchema creates all tables if they don't already exist.
func (s *Store) CreateSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS nodes (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    description_seed TEXT NOT NULL DEFAULT '',
    region TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS edges (
    from_node TEXT NOT NULL REFERENCES nodes(id),
    to_node TEXT NOT NULL REFERENCES nodes(id),
    from_direction TEXT NOT NULL CHECK(from_direction IN ('north','south','east','west','up','down')),
    to_direction TEXT NOT NULL CHECK(to_direction IN ('north','south','east','west','up','down')),
    type TEXT NOT NULL DEFAULT 'open',
    PRIMARY KEY (from_node, from_direction),
    UNIQUE (to_node, to_direction)
);

CREATE TABLE IF NOT EXISTS meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS enemy_templates (
    id TEXT PRIMARY KEY,
    name TEXT,
    hp INTEGER,
    attack TEXT,
    ai TEXT
);

CREATE TABLE IF NOT EXISTS item_templates (
    id TEXT PRIMARY KEY,
    name TEXT,
    type TEXT,
    damage TEXT,
    weight REAL,
    value INTEGER,
    description TEXT
);

CREATE TABLE IF NOT EXISTS spell_templates (
    id TEXT PRIMARY KEY,
    name TEXT,
    mp INTEGER,
    effect TEXT,
    power TEXT,
    classes TEXT
);

CREATE TABLE IF NOT EXISTS room_items (
    node_id TEXT REFERENCES nodes(id),
    item_id TEXT REFERENCES item_templates(id),
    PRIMARY KEY (node_id, item_id)
);

CREATE TABLE IF NOT EXISTS room_enemies (
    node_id TEXT REFERENCES nodes(id),
    enemy_id TEXT REFERENCES enemy_templates(id),
    PRIMARY KEY (node_id, enemy_id)
);`
	if _, err := s.db.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	return nil
}

// SaveGraph persists a Graph to the database, replacing any existing data.
func (s *Store) SaveGraph(g *Graph) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Clear existing data.
	for _, table := range []string{"room_enemies", "room_items", "edges", "nodes", "meta"} {
		if _, err := tx.Exec("DELETE FROM " + table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	// Save meta.
	metaStmt, err := tx.Prepare("INSERT INTO meta (key, value) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("prepare meta insert: %w", err)
	}
	defer metaStmt.Close()
	for k, v := range g.Meta {
		if _, err := metaStmt.Exec(k, v); err != nil {
			return fmt.Errorf("save meta %s: %w", k, err)
		}
	}
	if _, err := metaStmt.Exec("start", g.Start); err != nil {
		return fmt.Errorf("save start: %w", err)
	}

	// Save nodes.
	nodeStmt, err := tx.Prepare("INSERT INTO nodes (id, name, description_seed, region) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare node insert: %w", err)
	}
	defer nodeStmt.Close()

	for id, node := range g.Nodes {
		name := ""
		region := ""
		if node.Meta != nil {
			name = node.Meta["name"]
			region = node.Meta["region"]
		}
		if _, err := nodeStmt.Exec(id, name, "", region); err != nil {
			return fmt.Errorf("insert node %s: %w", id, err)
		}
	}

	// Save edges.
	edgeStmt, err := tx.Prepare("INSERT INTO edges (from_node, to_node, from_direction, to_direction, type) VALUES (?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare edge insert: %w", err)
	}
	defer edgeStmt.Close()

	for _, e := range g.Edges {
		// Store one row per logical edge. Bidirectionality is reconstructed
		// at the application layer (toScenario / buildConnectionMap).
		if _, err := edgeStmt.Exec(e.From, e.To, string(e.FromDir), string(e.ToDir), e.Type); err != nil {
			return fmt.Errorf("insert edge %s→%s: %w", e.From, e.To, err)
		}
	}

	return tx.Commit()
}

// LoadGraph reads a Graph from the database.
func (s *Store) LoadGraph() (*Graph, error) {
	// Read start node.
	var start string
	err := s.db.QueryRow("SELECT value FROM meta WHERE key = 'start'").Scan(&start)
	if err != nil {
		return nil, fmt.Errorf("read start: %w", err)
	}

	g := NewGraph(start)

	// Read all meta keys.
	metaRows, err := s.db.Query("SELECT key, value FROM meta WHERE key != 'start'")
	if err != nil {
		return nil, fmt.Errorf("query meta: %w", err)
	}
	defer metaRows.Close()
	for metaRows.Next() {
		var k, v string
		if err := metaRows.Scan(&k, &v); err != nil {
			return nil, fmt.Errorf("scan meta: %w", err)
		}
		g.Meta[k] = v
	}
	if err := metaRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate meta: %w", err)
	}

	// Read nodes.
	rows, err := s.db.Query("SELECT id, name, region FROM nodes")
	if err != nil {
		return nil, fmt.Errorf("query nodes: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, name, region string
		if err := rows.Scan(&id, &name, &region); err != nil {
			return nil, fmt.Errorf("scan node: %w", err)
		}
		meta := map[string]string{"name": name}
		if region != "" {
			meta["region"] = region
		}
		if err := g.AddNode(id, meta); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nodes: %w", err)
	}

	// Read all edges. Each logical edge is stored as one row.
	edgeRows, err := s.db.Query(`
		SELECT from_node, to_node, from_direction, to_direction, type FROM edges
	`)
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}
	defer edgeRows.Close()

	for edgeRows.Next() {
		var from, to, fromDir, toDir, edgeType string
		if err := edgeRows.Scan(&from, &to, &fromDir, &toDir, &edgeType); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		g.Edges = append(g.Edges, Edge{
			From: from, To: to,
			FromDir: Direction(fromDir), ToDir: Direction(toDir),
			Type: edgeType,
		})
	}
	if err := edgeRows.Err(); err != nil {
		return nil, fmt.Errorf("iterate edges: %w", err)
	}
	return g, nil
}
