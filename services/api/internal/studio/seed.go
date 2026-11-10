package studio

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
)

// SeedName and SeedVersion identify the canonical library shipped in seed/.
const (
	SeedName    = "restick.seed.v1"
	SeedVersion = 1
)

//go:embed seed/groups.json seed/fragments.json seed/palettes.json
var seedFS embed.FS

type seedGroups struct {
	Schema string      `json:"schema"`
	Groups []seedGroup `json:"groups"`
}

type seedGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

type seedFragments struct {
	Schema    string         `json:"schema"`
	Fragments []seedFragment `json:"fragments"`
}

type seedFragment struct {
	ID      string `json:"id"`
	GroupID string `json:"group_id"`
	Name    string `json:"name"`
	NameEn  string `json:"name_en"`
	Prompt  string `json:"prompt"`
	Notes   string `json:"notes"`
	Source  string `json:"source"`
	Scope   string `json:"scope"`
}

type seedPalettes struct {
	Schema   string        `json:"schema"`
	GroupID  string        `json:"group_id"`
	Palettes []seedPalette `json:"palettes"`
}

type seedPalette struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	NameEn    string `json:"name_en"`
	Prompt    string `json:"prompt"`
	Primary   string `json:"primary"`
	Secondary string `json:"secondary"`
	Accent    string `json:"accent"`
	Line      string `json:"line"`
}

// Seed installs the canonical library exactly once. Restarting with the same
// seed file leaves every operator edit in place.
func (s *Service) Seed(ctx context.Context) error {
	var applied int
	err := s.db.QueryRowContext(ctx, `SELECT version FROM seed_versions WHERE name = ?`, SeedName).Scan(&applied)
	switch {
	case err == nil:
		return nil
	case !errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("studio: read seed version: %w", err)
	}

	groups, fragments, palettes, checksum, err := loadSeed()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, group := range groups {
		if _, err := tx.ExecContext(ctx, `INSERT INTO fragment_groups (id, name, description, sort_order)
			VALUES (?, ?, ?, ?)`, group.ID, group.Name, group.Description, group.SortOrder); err != nil {
			return fmt.Errorf("studio: seed group %s: %w", group.ID, err)
		}
	}
	for _, fragment := range fragments {
		if _, err := tx.ExecContext(ctx, `INSERT INTO fragments
			(id, group_id, name, name_en, prompt, notes, source, scope)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			fragment.ID, fragment.GroupID, fragment.Name, fragment.NameEn,
			fragment.Prompt, fragment.Notes, fragment.Source, fragment.Scope); err != nil {
			return fmt.Errorf("studio: seed fragment %s: %w", fragment.ID, err)
		}
	}
	for _, palette := range palettes {
		if _, err := tx.ExecContext(ctx, `INSERT INTO fragments
			(id, group_id, name, name_en, prompt, source, scope,
			 primary_color, secondary_color, accent_color, line_color)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			palette.ID, paletteGroupID, palette.Name, palette.NameEn, palette.Prompt,
			"palette-recipes.md", string(ScopePack),
			palette.Primary, palette.Secondary, palette.Accent, palette.Line); err != nil {
			return fmt.Errorf("studio: seed palette %s: %w", palette.ID, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO seed_versions (name, version, checksum_sha256)
		VALUES (?, ?, ?)`, SeedName, SeedVersion, checksum); err != nil {
		return fmt.Errorf("studio: record seed version: %w", err)
	}
	return tx.Commit()
}

// paletteGroupID is the stable group that holds the palette fragments.
const paletteGroupID = "palette"

// loadSeed reads and validates the embedded canonical library. IDs, group
// references, scopes and palette colours are proved here so a bad edit cannot
// reach the database.
func loadSeed() ([]seedGroup, []seedFragment, []seedPalette, string, error) {
	groupBytes, err := seedFS.ReadFile("seed/groups.json")
	if err != nil {
		return nil, nil, nil, "", err
	}
	fragmentBytes, err := seedFS.ReadFile("seed/fragments.json")
	if err != nil {
		return nil, nil, nil, "", err
	}
	paletteBytes, err := seedFS.ReadFile("seed/palettes.json")
	if err != nil {
		return nil, nil, nil, "", err
	}

	var groups seedGroups
	var fragments seedFragments
	var palettes seedPalettes
	for _, doc := range []struct {
		name   string
		data   []byte
		into   any
		schema *string
	}{
		{"groups", groupBytes, &groups, &groups.Schema},
		{"fragments", fragmentBytes, &fragments, &fragments.Schema},
		{"palettes", paletteBytes, &palettes, &palettes.Schema},
	} {
		if err := json.Unmarshal(doc.data, doc.into); err != nil {
			return nil, nil, nil, "", fmt.Errorf("studio: decode seed %s: %w", doc.name, err)
		}
		if *doc.schema != SeedName {
			return nil, nil, nil, "", fmt.Errorf("studio: seed %s has schema %q, want %q", doc.name, *doc.schema, SeedName)
		}
	}
	if palettes.GroupID != paletteGroupID {
		return nil, nil, nil, "", fmt.Errorf("studio: palette seed group is %q, want %q", palettes.GroupID, paletteGroupID)
	}

	known := make(map[string]bool, len(groups.Groups))
	for _, group := range groups.Groups {
		switch {
		case group.ID == "" || group.Name == "":
			return nil, nil, nil, "", fmt.Errorf("studio: seed group %q needs an id and a name", group.ID)
		case known[group.ID]:
			return nil, nil, nil, "", fmt.Errorf("studio: duplicate seed group %q", group.ID)
		}
		known[group.ID] = true
	}
	if !known[paletteGroupID] {
		return nil, nil, nil, "", fmt.Errorf("studio: seed groups are missing the %q group", paletteGroupID)
	}

	ids := make(map[string]bool, len(fragments.Fragments)+len(palettes.Palettes))
	for _, fragment := range fragments.Fragments {
		switch {
		case fragment.ID == "" || fragment.Name == "" || fragment.Prompt == "":
			return nil, nil, nil, "", fmt.Errorf("studio: seed fragment %q needs an id, name and prompt", fragment.ID)
		case ids[fragment.ID]:
			return nil, nil, nil, "", fmt.Errorf("studio: duplicate seed fragment %q", fragment.ID)
		case !known[fragment.GroupID]:
			return nil, nil, nil, "", fmt.Errorf("studio: seed fragment %q references unknown group %q", fragment.ID, fragment.GroupID)
		case !validScope(Scope(fragment.Scope)):
			return nil, nil, nil, "", fmt.Errorf("studio: seed fragment %q has scope %q", fragment.ID, fragment.Scope)
		}
		ids[fragment.ID] = true
	}
	for _, palette := range palettes.Palettes {
		if palette.ID == "" || palette.Name == "" || palette.Prompt == "" {
			return nil, nil, nil, "", fmt.Errorf("studio: seed palette %q needs an id, name and prompt", palette.ID)
		}
		if ids[palette.ID] {
			return nil, nil, nil, "", fmt.Errorf("studio: duplicate seed fragment %q", palette.ID)
		}
		for _, color := range []string{palette.Primary, palette.Secondary, palette.Accent, palette.Line} {
			if !colorPattern.MatchString(color) {
				return nil, nil, nil, "", fmt.Errorf("studio: seed palette %q has a colour outside #RRGGBB", palette.ID)
			}
		}
		ids[palette.ID] = true
	}

	sum := sha256.New()
	for _, data := range [][]byte{groupBytes, fragmentBytes, paletteBytes} {
		sum.Write(data)
	}
	return groups.Groups, fragments.Fragments, palettes.Palettes, hex.EncodeToString(sum.Sum(nil)), nil
}
