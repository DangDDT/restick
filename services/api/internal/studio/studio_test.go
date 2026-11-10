package studio

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"math/rand"
	"strings"
	"testing"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/storage"
)

func newTestService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	dir := t.TempDir()
	db, err := storage.Open(context.Background(), dir)
	if err != nil {
		t.Fatalf("storage.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	files, err := assets.NewStore(dir)
	if err != nil {
		t.Fatalf("assets.NewStore: %v", err)
	}
	return NewService(db, files), db
}

func mustGroup(t *testing.T, service *Service, name string, order int) Group {
	t.Helper()
	group, err := service.CreateGroup(context.Background(), GroupDraft{Name: name, SortOrder: order})
	if err != nil {
		t.Fatalf("CreateGroup(%s): %v", name, err)
	}
	return group
}

func mustFragment(t *testing.T, service *Service, groupID, prompt string, scope Scope) Fragment {
	t.Helper()
	fragment, err := service.CreateFragment(context.Background(), FragmentDraft{
		GroupID: groupID,
		Name:    prompt,
		NameEn:  prompt,
		Prompt:  prompt,
		Scope:   scope,
	})
	if err != nil {
		t.Fatalf("CreateFragment(%s): %v", prompt, err)
	}
	return fragment
}

func mustPalette(t *testing.T, service *Service, groupID string, colors Colors) Fragment {
	t.Helper()
	fragment, err := service.CreateFragment(context.Background(), FragmentDraft{
		GroupID: groupID,
		Name:    "Bảng thử",
		NameEn:  "Test palette",
		Prompt:  "a warm test palette with cool accents",
		Scope:   ScopePack,
		Colors:  &colors,
	})
	if err != nil {
		t.Fatalf("CreateFragment palette: %v", err)
	}
	return fragment
}

// sheetDraft builds a valid 2x2 draft with the given selections.
func sheetDraft(fragmentIDs []string, groupIDs []string) Draft {
	return Draft{
		Name:        "Bộ thử",
		GroupIDs:    groupIDs,
		FragmentIDs: fragmentIDs,
		FreeText:    "a small seaside bundle",
		Rows:        2,
		Columns:     2,
		Cells: []Cell{
			{Action: "waving", Expression: "open smile", FreeText: "near a wooden sign"},
			{Action: "sitting", Expression: "closed eyes", FragmentIDs: nil},
			{Action: "walking", Expression: "flat mouth"},
			{Action: "jumping", Expression: "wide grin"},
		},
		OutlineWidth: 4,
	}
}

func TestArchivedFragmentDoesNotChangeSavedPrompt(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	group := mustGroup(t, service, "Nhận dạng", 10)
	scarf := mustFragment(t, service, group.ID, "a red scarf around the neck", ScopeIdentity)

	draft := sheetDraft([]string{scarf.ID}, []string{group.ID})
	saved, err := service.SaveDesign(ctx, draft)
	if err != nil {
		t.Fatalf("SaveDesign: %v", err)
	}
	if !strings.Contains(saved.Prompt, "a red scarf around the neck") {
		t.Fatalf("saved prompt does not carry the fragment text:\n%s", saved.Prompt)
	}

	if _, err := service.UpdateFragment(ctx, scarf.ID, FragmentDraft{
		GroupID: group.ID,
		Name:    "Khăn xanh",
		NameEn:  "Blue scarf",
		Prompt:  "a bright blue scarf around the neck",
		Scope:   ScopeIdentity,
	}); err != nil {
		t.Fatalf("UpdateFragment: %v", err)
	}
	archived, err := service.ArchiveFragment(ctx, scarf.ID)
	if err != nil {
		t.Fatalf("ArchiveFragment: %v", err)
	}
	if !archived.Archived {
		t.Fatal("fragment was not archived")
	}

	reloaded, err := service.GetDesign(ctx, saved.ID)
	if err != nil {
		t.Fatalf("GetDesign: %v", err)
	}
	if reloaded.Prompt != saved.Prompt {
		t.Fatalf("prompt changed after library edit:\nbefore: %s\nafter:  %s", saved.Prompt, reloaded.Prompt)
	}
	if strings.Contains(reloaded.Prompt, "blue scarf") {
		t.Fatalf("archived/edited fragment leaked into the saved prompt:\n%s", reloaded.Prompt)
	}
	if len(reloaded.Snapshot.Selections) != 1 || reloaded.Snapshot.Selections[0].Prompt != "a red scarf around the neck" {
		t.Fatalf("snapshot does not freeze the original text: %+v", reloaded.Snapshot.Selections)
	}
	if reloaded.Snapshot.SchemaVersion != 1 {
		t.Fatalf("snapshot schema_version = %d, want 1", reloaded.Snapshot.SchemaVersion)
	}

	// The library now refuses the archived fragment for new compiles.
	if _, err := service.Compile(ctx, draft); !errors.Is(err, ErrInvalid) {
		t.Fatalf("Compile with archived fragment = %v, want ErrInvalid", err)
	}
}

func TestSeedRestartPreservesUserEdits(t *testing.T) {
	service, db := newTestService(t)
	ctx := context.Background()
	if err := service.Seed(ctx); err != nil {
		t.Fatalf("Seed: %v", err)
	}

	count := func(table string) int {
		t.Helper()
		var total int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&total); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		return total
	}
	if got := count("fragment_groups"); got != 33 {
		t.Fatalf("seeded groups = %d, want 33", got)
	}
	if got := count("fragments"); got != 778 {
		t.Fatalf("seeded fragments = %d, want 754 visual+context and 24 palettes", got)
	}

	seeded, err := service.GetFragment(ctx, "v-d01-001")
	if err != nil {
		t.Fatalf("GetFragment(v-d01-001): %v", err)
	}
	if seeded.Name == "" || seeded.Prompt == "" || seeded.Scope != ScopeIdentity {
		t.Fatalf("seeded fragment is incomplete: %+v", seeded)
	}
	if _, err := service.UpdateFragment(ctx, seeded.ID, FragmentDraft{
		GroupID: seeded.GroupID,
		Name:    "Người tí hon (đã sửa)",
		NameEn:  "Tiny human edited",
		Prompt:  "the operator rewrote this prompt",
		Scope:   ScopeIdentity,
	}); err != nil {
		t.Fatalf("UpdateFragment: %v", err)
	}

	custom := mustGroup(t, service, "Nhóm riêng của tôi", 999)
	mustFragment(t, service, custom.ID, "a brass telescope on a strap", ScopePack)
	before := count("fragments")

	// A restart re-runs init against the same database.
	restarted := NewService(db, nil)
	if err := restarted.Seed(ctx); err != nil {
		t.Fatalf("Seed on restart: %v", err)
	}
	if got := count("fragments"); got != before {
		t.Fatalf("restart changed fragment count: %d -> %d", before, got)
	}
	if got := count("fragment_groups"); got != 34 {
		t.Fatalf("restart changed group count: %d, want 33 seeded + 1 custom", got)
	}
	edited, err := restarted.GetFragment(ctx, "v-d01-001")
	if err != nil {
		t.Fatalf("GetFragment after restart: %v", err)
	}
	if edited.Prompt != "the operator rewrote this prompt" || edited.Name != "Người tí hon (đã sửa)" {
		t.Fatalf("restart overwrote the operator edit: %+v", edited)
	}
	var seeds int
	if err := db.QueryRow(`SELECT COUNT(*) FROM seed_versions WHERE name = ?`, SeedName).Scan(&seeds); err != nil {
		t.Fatalf("read seed_versions: %v", err)
	}
	if seeds != 1 {
		t.Fatalf("seed_versions rows = %d, want 1", seeds)
	}

	// Seeded palettes stay selectable and compile with their four roles.
	palette, err := restarted.GetFragment(ctx, "p-001")
	if err != nil {
		t.Fatalf("GetFragment(p-001): %v", err)
	}
	if palette.Colors == nil || palette.Colors.Primary != "#6F4E37" {
		t.Fatalf("seeded palette colours = %+v", palette.Colors)
	}
	compiled, err := restarted.Compile(ctx, sheetDraft(nil, nil))
	if err != nil {
		t.Fatalf("Compile after seed: %v", err)
	}
	if !strings.Contains(compiled.Text, "Sheet: 2 rows by 2 columns") {
		t.Fatalf("compiled prompt missing sheet line:\n%s", compiled.Text)
	}
}

func TestPinnedSelectionsAndCellTextSurviveSuggestion(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	identity := mustGroup(t, service, "Nhận dạng", 10)
	style := mustGroup(t, service, "Phong cách", 20)
	unused := mustGroup(t, service, "Không tham gia", 30)

	pinned := mustFragment(t, service, identity.ID, "a fixed red scarf", ScopeIdentity)
	mustFragment(t, service, identity.ID, "a fixed green hat", ScopeIdentity)
	var styleIDs []string
	for _, prompt := range []string{"flat colour", "soft gradient", "hatched shadows"} {
		fragment := mustFragment(t, service, style.ID, prompt, ScopeIdentity)
		styleIDs = append(styleIDs, fragment.ID)
	}
	unusedFragment := mustFragment(t, service, unused.ID, "never injected", ScopePack)

	cellFragment := mustFragment(t, service, style.ID, "per-sticker prop: a paper crane", ScopeSticker)
	draft := sheetDraft([]string{pinned.ID, styleIDs[0]}, []string{identity.ID, style.ID})
	draft.PinnedFragmentIDs = []string{pinned.ID}
	draft.Cells[1].FragmentIDs = []string{cellFragment.ID}
	draft.Cells[1].FreeText = "keep this cell text exactly"

	for seed := int64(0); seed < 25; seed++ {
		suggested, err := service.Suggest(ctx, draft, rand.New(rand.NewSource(seed)))
		if err != nil {
			t.Fatalf("Suggest(seed %d): %v", seed, err)
		}
		if suggested.FreeText != draft.FreeText {
			t.Fatalf("suggestion overwrote free text: %q", suggested.FreeText)
		}
		if len(suggested.Cells) != len(draft.Cells) {
			t.Fatalf("suggestion changed the cell list: %d -> %d", len(draft.Cells), len(suggested.Cells))
		}
		for index := range draft.Cells {
			if suggested.Cells[index].Action != draft.Cells[index].Action ||
				suggested.Cells[index].Expression != draft.Cells[index].Expression ||
				suggested.Cells[index].FreeText != draft.Cells[index].FreeText {
				t.Fatalf("suggestion changed cell %d text: %+v", index, suggested.Cells[index])
			}
			if len(suggested.Cells[index].FragmentIDs) != len(draft.Cells[index].FragmentIDs) {
				t.Fatalf("suggestion changed cell %d fragments: %v", index, suggested.Cells[index].FragmentIDs)
			}
		}
		if len(suggested.PinnedFragmentIDs) != 1 || suggested.PinnedFragmentIDs[0] != pinned.ID {
			t.Fatalf("suggestion dropped the pin: %v", suggested.PinnedFragmentIDs)
		}
		if len(suggested.GroupIDs) != 2 {
			t.Fatalf("suggestion injected a non-participating group: %v", suggested.GroupIDs)
		}
		if len(suggested.FragmentIDs) != 2 {
			t.Fatalf("suggestion should pick one fragment per participating group: %v", suggested.FragmentIDs)
		}

		byID := map[string]Fragment{}
		for _, id := range suggested.FragmentIDs {
			fragment, err := service.GetFragment(ctx, id)
			if err != nil {
				t.Fatalf("GetFragment(%s): %v", id, err)
			}
			byID[id] = fragment
		}
		if _, ok := byID[pinned.ID]; !ok {
			t.Fatalf("pinned selection was replaced: %v", suggested.FragmentIDs)
		}
		var styleCount, identityCount int
		for _, fragment := range byID {
			switch fragment.GroupID {
			case style.ID:
				styleCount++
				if fragment.ID == unusedFragment.ID {
					t.Fatal("unrelated fragment injected")
				}
			case identity.ID:
				identityCount++
			default:
				t.Fatalf("suggestion used a group that is not participating: %s", fragment.GroupID)
			}
		}
		if styleCount != 1 || identityCount != 1 {
			t.Fatalf("expected one selection per participating group, got %v", byID)
		}
	}
}

func TestCompileRejectsCellCountAndOutlineRange(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	valid := sheetDraft(nil, nil)
	if _, err := service.Compile(ctx, valid); err != nil {
		t.Fatalf("valid draft must compile: %v", err)
	}
	// Boundaries 0 and 16 px, 1x1 and 8x8 grids are accepted.
	smallest := Draft{Rows: 1, Columns: 1, Cells: []Cell{{Action: "waving"}}, OutlineWidth: 0}
	if _, err := service.Compile(ctx, smallest); err != nil {
		t.Fatalf("1x1 draft with a zero outline must compile: %v", err)
	}
	widest := Draft{Rows: 1, Columns: 1, Cells: []Cell{{Action: "waving"}}, OutlineWidth: 16, OutlineColor: "#00ff88"}
	compiled, err := service.Compile(ctx, widest)
	if err != nil {
		t.Fatalf("16 px outline must compile: %v", err)
	}
	if !strings.Contains(compiled.Text, "16 px #00FF88 outline") {
		t.Fatalf("outline line missing:\n%s", compiled.Text)
	}
	grid := Draft{Rows: 8, Columns: 8, Cells: make([]Cell, 64), OutlineWidth: 4}
	for index := range grid.Cells {
		grid.Cells[index] = Cell{Action: "standing"}
	}
	if _, err := service.Compile(ctx, grid); err != nil {
		t.Fatalf("8x8 draft must compile: %v", err)
	}

	cases := map[string]Draft{
		"fewer cells than the grid": func() Draft {
			draft := sheetDraft(nil, nil)
			draft.Cells = draft.Cells[:3]
			return draft
		}(),
		"more cells than the grid": func() Draft {
			draft := sheetDraft(nil, nil)
			draft.Cells = append(draft.Cells, Cell{Action: "extra"})
			return draft
		}(),
		"zero rows":          {Rows: 0, Columns: 1, Cells: []Cell{{Action: "waving"}}, OutlineWidth: 4},
		"nine columns":       {Rows: 1, Columns: 9, Cells: make([]Cell, 9), OutlineWidth: 4},
		"outline negative":   {Rows: 1, Columns: 1, Cells: []Cell{{Action: "waving"}}, OutlineWidth: -1},
		"outline too wide":   {Rows: 1, Columns: 1, Cells: []Cell{{Action: "waving"}}, OutlineWidth: 17},
		"bad outline colour": {Rows: 1, Columns: 1, Cells: []Cell{{Action: "waving"}}, OutlineWidth: 4, OutlineColor: "#FFF"},
		"empty cell":         {Rows: 1, Columns: 1, Cells: []Cell{{}}, OutlineWidth: 4},
	}
	for name, draft := range cases {
		if _, err := service.Compile(ctx, draft); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: Compile error = %v, want ErrInvalid", name, err)
		}
	}

	unknown := sheetDraft([]string{"not-a-fragment"}, nil)
	if _, err := service.Compile(ctx, unknown); !errors.Is(err, ErrInvalid) {
		t.Errorf("unknown fragment: Compile error = %v, want ErrInvalid", err)
	}
	unknownPalette := sheetDraft(nil, nil)
	unknownPalette.PaletteID = "p-999"
	if _, err := service.Compile(ctx, unknownPalette); !errors.Is(err, ErrInvalid) {
		t.Errorf("unknown palette: Compile error = %v, want ErrInvalid", err)
	}
}

func TestCompileDoesNotMixWhiteAndTransparentRequirements(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	group := mustGroup(t, service, "Bối cảnh", 10)
	mustFragment(t, service, group.ID, "a windowsill with a small plant", ScopePack)

	compiled, err := service.Compile(ctx, sheetDraft(nil, []string{group.ID}))
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	text := strings.ToLower(compiled.Text)
	if !strings.Contains(text, "white background") {
		t.Fatalf("prompt does not ask for the flat white sheet background:\n%s", compiled.Text)
	}
	for _, banned := range []string{"transparent", "alpha"} {
		if strings.Contains(text, banned) {
			t.Fatalf("prompt mixes a second background requirement (%q):\n%s", banned, compiled.Text)
		}
	}
	// The technical block comes last and states the outline once.
	technical := strings.Index(compiled.Text, "Technical:")
	if technical < 0 || technical < strings.Index(compiled.Text, "Stickers:") {
		t.Fatalf("technical block must follow the cells:\n%s", compiled.Text)
	}
}

func TestGroupDeleteRejectsMembers(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	withMember := mustGroup(t, service, "Có mảnh", 10)
	fragment := mustFragment(t, service, withMember.ID, "a stubby pencil", ScopePack)
	empty := mustGroup(t, service, "Trống", 20)

	if err := service.DeleteGroup(ctx, withMember.ID); !errors.Is(err, ErrGroupNotEmpty) {
		t.Fatalf("DeleteGroup with member = %v, want ErrGroupNotEmpty", err)
	}
	if _, err := service.GetGroup(ctx, withMember.ID); err != nil {
		t.Fatalf("group must survive a refused delete: %v", err)
	}
	if _, err := service.GetFragment(ctx, fragment.ID); err != nil {
		t.Fatalf("fragment must survive a refused delete: %v", err)
	}
	if _, err := service.ArchiveFragment(ctx, fragment.ID); err != nil {
		t.Fatalf("ArchiveFragment: %v", err)
	}
	if err := service.DeleteGroup(ctx, withMember.ID); !errors.Is(err, ErrGroupNotEmpty) {
		t.Fatalf("DeleteGroup with an archived member = %v, want ErrGroupNotEmpty", err)
	}

	if err := service.DeleteGroup(ctx, empty.ID); err != nil {
		t.Fatalf("DeleteGroup on an empty group: %v", err)
	}
	if _, err := service.GetGroup(ctx, empty.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted group read = %v, want ErrNotFound", err)
	}
	if err := service.DeleteGroup(ctx, empty.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second delete = %v, want ErrNotFound", err)
	}
}

func TestPaletteAndCustomGroupsCompile(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	custom := mustGroup(t, service, "Nhóm tự tạo", 900)
	telescope := mustFragment(t, service, custom.ID, "a brass telescope on a leather strap", ScopePack)
	paletteGroup := mustGroup(t, service, "Bảng màu của tôi", 950)
	palette := mustPalette(t, service, paletteGroup.ID, Colors{
		Primary: "#123456", Secondary: "#ABCDEF", Accent: "#FF8800", Line: "#101010",
	})

	draft := sheetDraft([]string{telescope.ID}, []string{custom.ID})
	draft.PaletteID = palette.ID
	compiled, err := service.Compile(ctx, draft)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	for _, want := range []string{
		"a brass telescope on a leather strap",
		"a warm test palette with cool accents",
		"Primary #123456, secondary #ABCDEF, accent #FF8800, line #101010",
	} {
		if !strings.Contains(compiled.Text, want) {
			t.Fatalf("compiled prompt is missing %q:\n%s", want, compiled.Text)
		}
	}
	if compiled.Snapshot.Palette == nil || compiled.Snapshot.Palette.FragmentID != palette.ID {
		t.Fatalf("snapshot palette = %+v", compiled.Snapshot.Palette)
	}
	if len(compiled.Snapshot.Selections) != 1 || compiled.Snapshot.Selections[0].GroupName != "Nhóm tự tạo" {
		t.Fatalf("snapshot selections = %+v", compiled.Snapshot.Selections)
	}

	// A fragment without palette colours cannot be used as the palette.
	plain := mustFragment(t, service, custom.ID, "a plain strap", ScopePack)
	draft.PaletteID = plain.ID
	if _, err := service.Compile(ctx, draft); !errors.Is(err, ErrInvalid) {
		t.Fatalf("non-palette id as palette = %v, want ErrInvalid", err)
	}
	draft.PaletteID = palette.ID
}

func TestReferenceAssetStaysPrivateAndFeedsCharacterSnapshot(t *testing.T) {
	service, db := newTestService(t)
	ctx := context.Background()
	raw := pngBytes(t)

	asset, err := service.SaveReferenceAsset(ctx, bytes.NewReader(raw), "reference.png")
	if err != nil {
		t.Fatalf("SaveReferenceAsset: %v", err)
	}
	meta, file, err := service.OpenReferenceAsset(ctx, asset.ID)
	if err != nil {
		t.Fatalf("OpenReferenceAsset: %v", err)
	}
	defer file.Close()
	if meta.ContentType != "image/png" || meta.Width != 4 || meta.Height != 4 || meta.ByteSize != int64(len(raw)) {
		t.Fatalf("reference metadata = %+v", meta)
	}
	read, err := io.ReadAll(file)
	if err != nil || !bytes.Equal(read, raw) {
		t.Fatalf("reference bytes mismatch: %v", err)
	}

	character, err := service.CreateCharacter(ctx, CharacterDraft{
		Name:             "Bé Mực",
		Description:      "a small ink-drop character",
		IdentityNotes:    "one white hair streak, always a red scarf",
		RightsNotes:      "own artwork, no third-party reference",
		ReferenceAssetID: asset.ID,
	})
	if err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	draft := sheetDraft(nil, nil)
	draft.CharacterID = character.ID
	compiled, err := service.Compile(ctx, draft)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if !strings.Contains(compiled.Text, "Character: Bé Mực") ||
		!strings.Contains(compiled.Text, "a small ink-drop character") ||
		!strings.Contains(compiled.Text, "one white hair streak, always a red scarf") {
		t.Fatalf("identity block is incomplete:\n%s", compiled.Text)
	}
	if compiled.Snapshot.Character == nil || compiled.Snapshot.Character.ID != character.ID {
		t.Fatalf("snapshot character = %+v", compiled.Snapshot.Character)
	}

	if _, err := service.ArchiveCharacter(ctx, character.ID); err != nil {
		t.Fatalf("ArchiveCharacter: %v", err)
	}
	characters, err := service.ListCharacters(ctx)
	if err != nil {
		t.Fatalf("ListCharacters: %v", err)
	}
	if len(characters) != 0 {
		t.Fatalf("archived character still listed: %+v", characters)
	}
	// The asset row and its bytes stay readable for existing profiles.
	if _, err := service.GetReferenceAsset(ctx, asset.ID); err != nil {
		t.Fatalf("GetReferenceAsset: %v", err)
	}
	var stored int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reference_assets WHERE id = ?`, asset.ID).Scan(&stored); err != nil || stored != 1 {
		t.Fatalf("reference_assets row = %d (%v)", stored, err)
	}
	if _, err := service.CreateCharacter(ctx, CharacterDraft{Name: "No asset", ReferenceAssetID: "missing"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("character with unknown asset = %v, want ErrInvalid", err)
	}
}

func TestSaveDesignRewritesSnapshotOnUpdate(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	draft := sheetDraft(nil, nil)

	saved, err := service.SaveDesign(ctx, draft)
	if err != nil {
		t.Fatalf("SaveDesign: %v", err)
	}
	if saved.Draft.ID != saved.ID || saved.Rows != 2 || saved.Columns != 2 {
		t.Fatalf("stored draft does not echo the design: %+v", saved.Draft)
	}

	draft.ID = saved.ID
	draft.Cells[0].Action = "tiptoeing"
	draft.OutlineWidth = 0
	updated, err := service.SaveDesign(ctx, draft)
	if err != nil {
		t.Fatalf("SaveDesign update: %v", err)
	}
	if updated.ID != saved.ID {
		t.Fatalf("update created a new design: %s != %s", updated.ID, saved.ID)
	}
	if !strings.Contains(updated.Prompt, "tiptoeing") || strings.Contains(updated.Prompt, "waving") {
		t.Fatalf("prompt was not rebuilt:\n%s", updated.Prompt)
	}
	if updated.Snapshot.OutlineWidth != 0 || updated.Snapshot.Cells[0].Action != "tiptoeing" {
		t.Fatalf("snapshot was not rebuilt: %+v", updated.Snapshot)
	}
	if updated.CreatedAt.IsZero() || updated.UpdatedAt.Before(updated.CreatedAt) {
		t.Fatalf("timestamps look wrong: %s / %s", updated.CreatedAt, updated.UpdatedAt)
	}

	page, err := service.ListDesigns(ctx, DesignFilter{Query: "Bộ", Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListDesigns: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != saved.ID {
		t.Fatalf("ListDesigns = %+v", page)
	}
	if err := service.DeleteDesign(ctx, saved.ID); err != nil {
		t.Fatalf("DeleteDesign: %v", err)
	}
	if _, err := service.GetDesign(ctx, saved.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("deleted design read = %v, want ErrNotFound", err)
	}
}

func TestListFragmentsFiltersAndArchiveVisibility(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	props := mustGroup(t, service, "Đạo cụ", 10)
	culture := mustGroup(t, service, "Văn hoá", 20)

	first, err := service.CreateFragment(ctx, FragmentDraft{
		GroupID: props.ID, Name: "La bàn", NameEn: "Compass", Prompt: "a brass compass", Scope: ScopePack,
	})
	if err != nil {
		t.Fatalf("CreateFragment: %v", err)
	}
	if _, err := service.CreateFragment(ctx, FragmentDraft{
		GroupID: culture.ID, Name: "Chợ đêm", NameEn: "Night market", Prompt: "a night market stall", Scope: ScopeSticker,
	}); err != nil {
		t.Fatalf("CreateFragment: %v", err)
	}
	if _, err := service.ArchiveFragment(ctx, first.ID); err != nil {
		t.Fatalf("ArchiveFragment: %v", err)
	}

	cases := []struct {
		name   string
		filter FragmentFilter
		total  int
	}{
		{"all active", FragmentFilter{}, 1},
		{"include archived", FragmentFilter{IncludeArchived: true}, 2},
		{"vietnamese name", FragmentFilter{Query: "chợ"}, 1},
		{"english name", FragmentFilter{Query: "compass", IncludeArchived: true}, 1},
		{"group", FragmentFilter{GroupID: props.ID, IncludeArchived: true}, 1},
		{"scope", FragmentFilter{Scope: ScopeSticker}, 1},
		{"page size", FragmentFilter{IncludeArchived: true, PageSize: 1, Page: 2}, 2},
	}
	for _, testCase := range cases {
		page, err := service.ListFragments(ctx, testCase.filter)
		if err != nil {
			t.Fatalf("%s: ListFragments: %v", testCase.name, err)
		}
		if page.Total != testCase.total {
			t.Fatalf("%s: total = %d, want %d", testCase.name, page.Total, testCase.total)
		}
		if testCase.filter.PageSize == 1 && len(page.Items) != 1 {
			t.Fatalf("%s: page size was not applied: %d items", testCase.name, len(page.Items))
		}
	}
	if _, err := service.ListFragments(ctx, FragmentFilter{Scope: "nonsense"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown scope filter = %v, want ErrInvalid", err)
	}
}

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for x := range 4 {
		for y := range 4 {
			img.Set(x, y, color.RGBA{R: uint8(x * 40), G: uint8(y * 40), B: 200, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}
	return buffer.Bytes()
}
