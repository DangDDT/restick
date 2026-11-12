package studio

import (
	"context"
	"fmt"
	"math/rand" // math/rand v1: the approved Suggest signature is asked for a seeded *rand.Rand
	"sort"
	"strconv"
	"strings"
	"time"
)

// MaxCells is the largest accepted sheet (8 rows by 8 columns).
const MaxCells = 64

// DefaultOutlineColor is the outline colour used when a draft leaves it empty.
// DefaultOutlineWidth is the width the HTTP/UI layer preinitialises on a new
// draft; Compile keeps a plain int, so an explicit 0 stays "no outline" and is
// never silently replaced by this default.
const (
	DefaultOutlineColor = "#FFFFFF"
	DefaultOutlineWidth = 4
)

// Compile resolves a draft against the library and renders the English prompt.
// The returned snapshot freezes every fragment text used, so the prompt never
// depends on the library again once it is saved.
func (s *Service) Compile(ctx context.Context, draft Draft) (CompiledPrompt, error) {
	draft = normalizeDraft(draft)
	if err := validateDraft(draft); err != nil {
		return CompiledPrompt{}, err
	}
	ids := make([]string, 0, len(draft.FragmentIDs)+len(draft.PinnedFragmentIDs))
	ids = append(ids, draft.FragmentIDs...)
	ids = append(ids, draft.PinnedFragmentIDs...)
	for _, cell := range draft.Cells {
		ids = append(ids, cell.FragmentIDs...)
	}
	if draft.PaletteID != "" {
		ids = append(ids, draft.PaletteID)
	}
	fragments, err := s.loadFragments(ctx, ids)
	if err != nil {
		return CompiledPrompt{}, err
	}
	if err := validateCharacterIdentity(draft.CharacterID, fragments); err != nil {
		return CompiledPrompt{}, err
	}
	groups, err := s.groupMap(ctx, fragments)
	if err != nil {
		return CompiledPrompt{}, err
	}

	var character *Character
	if draft.CharacterID != "" {
		found, err := s.GetCharacter(ctx, draft.CharacterID)
		if err != nil {
			return CompiledPrompt{}, err
		}
		character = &found
	}
	if err := validateReference(draft, character); err != nil {
		return CompiledPrompt{}, err
	}

	var palette *SnapshotPalette
	if draft.PaletteID != "" {
		fragment, ok := fragments[draft.PaletteID]
		if !ok || fragment.Archived || fragment.Colors == nil {
			return CompiledPrompt{}, fmt.Errorf("%w: palette %q is missing, archived or has no colours", ErrInvalid, draft.PaletteID)
		}
		palette = &SnapshotPalette{
			FragmentID: fragment.ID,
			Name:       fragment.Name,
			Prompt:     fragment.Prompt,
			Primary:    fragment.Colors.Primary,
			Secondary:  fragment.Colors.Secondary,
			Accent:     fragment.Colors.Accent,
			Line:       fragment.Colors.Line,
		}
	}

	selections, err := selectionsFor(draft.FragmentIDs, fragments, groups)
	if err != nil {
		return CompiledPrompt{}, err
	}
	cells := make([]SnapshotCell, 0, len(draft.Cells))
	for index, cell := range draft.Cells {
		frozen, err := selectionsFor(cell.FragmentIDs, fragments, groups)
		if err != nil {
			return CompiledPrompt{}, err
		}
		cells = append(cells, SnapshotCell{
			Row:        index/draft.Columns + 1,
			Column:     index%draft.Columns + 1,
			Action:     cell.Action,
			Expression: cell.Expression,
			FreeText:   cell.FreeText,
			Label:      cell.Label,
			Fragments:  frozen,
		})
	}

	snapshot := Snapshot{
		SchemaVersion: 1,
		Selections:    selections,
		Palette:       palette,
		FreeText:      draft.FreeText,
		Rows:          draft.Rows,
		Columns:       draft.Columns,
		OutlineWidth:  draft.OutlineWidth,
		OutlineColor:  draft.OutlineColor,
		Cells:         cells,
		CompiledAt:    time.Now().UTC().Format(timestampLayout),
	}
	if character != nil {
		snapshot.Character = &SnapshotCharacter{
			ID:               character.ID,
			Name:             character.Name,
			Description:      character.Description,
			IdentityNotes:    character.IdentityNotes,
			ReferenceAssetID: draft.ReferenceAssetID,
		}
	}
	return CompiledPrompt{Text: renderPrompt(draft, snapshot), Snapshot: snapshot}, nil
}

func validateCharacterIdentity(characterID string, fragments map[string]Fragment) error {
	if characterID == "" {
		return nil
	}
	for _, fragment := range fragments {
		if fragment.Scope == ScopeIdentity {
			return fmt.Errorf("%w: character profile owns identity; remove identity fragment %q from the design", ErrInvalid, fragment.Name)
		}
	}
	return nil
}

// validateReference enforces the selected-image rule of spec §5.2: a design
// with a character needs one image that belongs to that character, and a design
// without a character must not carry one.
func validateReference(draft Draft, character *Character) error {
	switch {
	case character == nil && draft.ReferenceAssetID == "":
		return nil
	case character == nil:
		return fmt.Errorf("%w: reference %q needs a character", ErrInvalid, draft.ReferenceAssetID)
	case draft.ReferenceAssetID == "":
		return fmt.Errorf("%w: character %q needs a selected reference image", ErrInvalid, character.Name)
	}
	for _, id := range character.ReferenceAssetIDs {
		if id == draft.ReferenceAssetID {
			return nil
		}
	}
	return fmt.Errorf("%w: reference %q is not attached to character %q", ErrInvalid, draft.ReferenceAssetID, character.Name)
}

// Suggest re-draws the unpinned selections of the participating groups. Pinned
// selections, free text, cell content and every non-selection field are left
// exactly as they were.
func (s *Service) Suggest(ctx context.Context, draft Draft, rng *rand.Rand) (Draft, error) {
	draft = normalizeDraft(draft)
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	topLevel := make([]string, 0, len(draft.FragmentIDs)+len(draft.PinnedFragmentIDs))
	topLevel = append(topLevel, draft.FragmentIDs...)
	topLevel = append(topLevel, draft.PinnedFragmentIDs...)
	selectionIDs := topLevel
	if draft.CharacterID != "" {
		for _, cell := range draft.Cells {
			topLevel = append(topLevel, cell.FragmentIDs...)
		}
		if draft.PaletteID != "" {
			topLevel = append(topLevel, draft.PaletteID)
		}
	}
	known, err := s.loadFragments(ctx, topLevel)
	if err != nil {
		return Draft{}, err
	}
	if err := validateCharacterIdentity(draft.CharacterID, known); err != nil {
		return Draft{}, err
	}
	pinned := map[string]bool{}
	for _, id := range draft.PinnedFragmentIDs {
		fragment, ok := known[id]
		if !ok || fragment.Archived {
			return Draft{}, fmt.Errorf("%w: pinned fragment %q is missing or archived", ErrInvalid, id)
		}
		pinned[id] = true
	}
	participating := map[string]bool{}
	for _, id := range draft.GroupIDs {
		if err := s.requireGroup(ctx, id); err != nil {
			return Draft{}, err
		}
		participating[id] = true
	}
	for _, id := range selectionIDs {
		if fragment, ok := known[id]; ok {
			participating[fragment.GroupID] = true
		}
	}

	groupOrder, err := s.loadGroups(ctx, keys(participating))
	if err != nil {
		return Draft{}, err
	}
	candidates, err := s.candidates(ctx, keys(participating), draft.CharacterID != "")
	if err != nil {
		return Draft{}, err
	}

	pinnedByGroup := map[string]string{}
	for _, id := range draft.PinnedFragmentIDs {
		group := known[id].GroupID
		if _, taken := pinnedByGroup[group]; !taken {
			pinnedByGroup[group] = id
		}
	}

	ordered := make([]string, 0, len(participating))
	for id := range participating {
		ordered = append(ordered, id)
	}
	sort.Slice(ordered, func(i, j int) bool {
		left, right := groupOrder[ordered[i]], groupOrder[ordered[j]]
		if left.SortOrder != right.SortOrder {
			return left.SortOrder < right.SortOrder
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		return left.ID < right.ID
	})

	picked := make([]string, 0, len(ordered))
	for _, groupID := range ordered {
		if id, ok := pinnedByGroup[groupID]; ok {
			picked = append(picked, id)
			continue
		}
		pool := candidates[groupID]
		if len(pool) == 0 {
			continue
		}
		picked = append(picked, pool[rng.Intn(len(pool))])
	}
	draft.GroupIDs = ordered
	draft.FragmentIDs = picked
	return draft, nil
}

// validateDraft enforces the sheet limits of spec §5.3.
func validateDraft(draft Draft) error {
	switch {
	case draft.Rows < 1 || draft.Rows > 8:
		return fmt.Errorf("%w: rows must be 1-8", ErrInvalid)
	case draft.Columns < 1 || draft.Columns > 8:
		return fmt.Errorf("%w: columns must be 1-8", ErrInvalid)
	case draft.Rows*draft.Columns > MaxCells:
		return fmt.Errorf("%w: a sheet holds at most %d stickers", ErrInvalid, MaxCells)
	case len(draft.Cells) != draft.Rows*draft.Columns:
		return fmt.Errorf("%w: %d cells described but the grid needs %d", ErrInvalid, len(draft.Cells), draft.Rows*draft.Columns)
	case draft.OutlineWidth < 0 || draft.OutlineWidth > 16:
		return fmt.Errorf("%w: outline width must be 0-16 px", ErrInvalid)
	case !colorPattern.MatchString(draft.OutlineColor):
		return fmt.Errorf("%w: outline colour must be #RRGGBB", ErrInvalid)
	}
	for index, cell := range draft.Cells {
		if cell.Action == "" && cell.Expression == "" && cell.FreeText == "" && len(cell.FragmentIDs) == 0 {
			return fmt.Errorf("%w: cell %d is empty", ErrInvalid, index+1)
		}
	}
	return nil
}

func normalizeDraft(draft Draft) Draft {
	draft.Name = strings.TrimSpace(draft.Name)
	draft.CharacterID = strings.TrimSpace(draft.CharacterID)
	draft.ReferenceAssetID = strings.TrimSpace(draft.ReferenceAssetID)
	draft.PaletteID = strings.TrimSpace(draft.PaletteID)
	draft.FreeText = strings.TrimSpace(draft.FreeText)
	draft.OutlineColor = strings.ToUpper(strings.TrimSpace(draft.OutlineColor))
	if draft.OutlineColor == "" {
		draft.OutlineColor = DefaultOutlineColor
	}
	draft.GroupIDs = uniqueStrings(draft.GroupIDs)
	draft.FragmentIDs = uniqueStrings(draft.FragmentIDs)
	draft.PinnedFragmentIDs = uniqueStrings(draft.PinnedFragmentIDs)
	cells := make([]Cell, 0, len(draft.Cells))
	for _, cell := range draft.Cells {
		cells = append(cells, Cell{
			Action:      strings.TrimSpace(cell.Action),
			Expression:  strings.TrimSpace(cell.Expression),
			FragmentIDs: uniqueStrings(cell.FragmentIDs),
			FreeText:    strings.TrimSpace(cell.FreeText),
			Label:       strings.TrimSpace(cell.Label),
		})
	}
	draft.Cells = cells
	return draft
}

// renderPrompt follows spec §7: opening, identity or per-subject block, theme,
// style/palette, every cell with its drawing instruction and label, then the
// technical sheet instruction. It is written for an external image agent that
// sees only this text.
func renderPrompt(draft Draft, snapshot Snapshot) string {
	var b strings.Builder
	total := snapshot.Rows * snapshot.Columns
	hasLabels := false
	for _, cell := range snapshot.Cells {
		if cell.Label != "" {
			hasLabels = true
			break
		}
	}
	if character := snapshot.Character; character != nil {
		fmt.Fprintf(&b, "Task: draw one sticker sheet containing exactly %d separate sticker illustrations of the same character, laid out in a regular grid. Software will cut the sheet automatically, so the Technical rules below matter as much as the artwork.\n", total)
		fmt.Fprintf(&b, "Character: %s.", character.Name)
		if character.Description != "" {
			b.WriteString(" " + finishSentence(character.Description))
		}
		if character.IdentityNotes != "" {
			b.WriteString(" Keep these identity features identical on every sticker: " + character.IdentityNotes)
		}
		b.WriteString("\n")
		b.WriteString("Reference image: attach the reference image selected for this design to the request and use it as the identity source. It is private and is not linked in this prompt. Keep the same proportions, colours, face design and line weight on every sticker; only pose, expression and props change.\n")
	} else {
		fmt.Fprintf(&b, "Task: draw one sticker sheet containing exactly %d separate sticker illustrations, laid out in a regular grid. Software will cut the sheet automatically, so the Technical rules below matter as much as the artwork.\n", total)
		b.WriteString("Subjects: each sticker is described separately below and the stickers may show different subjects, objects or dishes, while line weight, rendering and presentation stay consistent across the whole sheet. When the Theme specifies a subject, use it for each sticker unless that sticker's drawing description explicitly specifies a different subject.\n")
	}

	identity := make([]string, 0, len(snapshot.Selections))
	style := make([]string, 0, len(snapshot.Selections))
	for _, selection := range snapshot.Selections {
		if selection.Scope == ScopeIdentity {
			identity = append(identity, selection.Prompt)
			continue
		}
		style = append(style, selection.Prompt)
	}
	if len(identity) > 0 {
		fmt.Fprintf(&b, "Identity: %s.\n", strings.Join(identity, "; "))
	}
	if snapshot.FreeText != "" {
		fmt.Fprintf(&b, "Theme: %s\n", finishSentence(snapshot.FreeText))
	}
	if len(style) > 0 {
		fmt.Fprintf(&b, "Style, colour and props: %s.\n", strings.Join(style, "; "))
	}
	if palette := snapshot.Palette; palette != nil {
		fmt.Fprintf(&b, "Palette: %s Primary %s, secondary %s, accent %s, line %s.\n",
			finishSentence(palette.Prompt), palette.Primary, palette.Secondary, palette.Accent, palette.Line)
		b.WriteString("Palette use: primary for the largest areas, secondary for supporting areas, accent only for small highlights and props, line for the artwork's own contour lines. Use flat fills; do not introduce unrelated dominant colours. The character's identity colours and each subject's natural colours take precedence: use the palette only in supporting areas, never to recolour the character or the subject to match it.\n")
	}
	fmt.Fprintf(&b, "Sheet: %d rows by %d columns, %d stickers in total, read left to right then top to bottom. Every sticker sits in its own equal-sized area of the sheet.\n",
		snapshot.Rows, snapshot.Columns, total)
	b.WriteString(layoutRules(snapshot.Rows, snapshot.Columns, hasLabels))
	if hasLabels {
		b.WriteString("Stickers: each numbered line gives a drawing instruction and the exact visible label to write on that sticker. Read the instruction as what to draw, never as text to write; instructions may be written in Vietnamese. Letter each quoted label exactly as written, keeping its Vietnamese spelling and accents, without translating, transliterating or rewording it.\n")
	} else {
		b.WriteString("Stickers: read each description as a drawing instruction, never as text to write on the image; it may be written in Vietnamese.\n")
	}
	for index, cell := range snapshot.Cells {
		fmt.Fprintf(&b, "%d. (row %d, column %d) %s\n", index+1, cell.Row, cell.Column, cellSentence(cell))
	}
	if hasLabels {
		b.WriteString("Label style: letter every visible label in one uniform, legible typeface sized to read at sticker size. Make the label part of its own sticker: put it on a solid coloured ribbon or banner that physically connects to the subject's drawing, so the cutter never separates letters from the picture. Vietnamese diacritics must sit on that solid fill, not float on the white background. Write only the labels listed above; never write the drawing descriptions, cell numbers, coordinates, group names or this prompt as text.\n")
	} else {
		b.WriteString("Text: this sheet has no visible labels, so do not write any text on the stickers.\n")
	}
	b.WriteString(outlineLine(draft.OutlineWidth, draft.OutlineColor))
	return strings.TrimRight(b.String(), " \n") + "\n"
}

func cellSentence(cell SnapshotCell) string {
	parts := make([]string, 0, 3+len(cell.Fragments))
	if cell.Action != "" {
		parts = append(parts, cell.Action)
	}
	if cell.Expression != "" {
		parts = append(parts, cell.Expression)
	}
	for _, fragment := range cell.Fragments {
		parts = append(parts, fragment.Prompt)
	}
	if cell.FreeText != "" {
		parts = append(parts, cell.FreeText)
	}
	for index, part := range parts {
		parts[index] = strings.TrimRight(strings.TrimSpace(part), ".")
	}
	drawing := strings.Join(parts, "; ") + "."
	if cell.Label == "" {
		return drawing + " Visible label: none; this sticker has no text."
	}
	return drawing + " Visible label (write exactly, with accents): " + strconv.Quote(cell.Label) + "."
}

// layoutRules states the extraction contract once, before the cells: the sheet
// is cut by software into an equal grid, so every rule maps to a way the cut
// fails (touching stickers, a second background, a box crossing a grid line).
// Only an unlabeled sheet bans all text; with labels, the label rule replaces
// that sentence so the prompt cannot contradict itself.
func layoutRules(rows, columns int, hasLabels bool) string {
	total := rows * columns
	d := gcd(columns, rows)
	var b strings.Builder
	b.WriteString("Technical (software cuts this sheet automatically, so follow every rule exactly):\n")
	fmt.Fprintf(&b, "- Output one single image with aspect ratio %d:%d (width:height), divided into %d rows by %d columns of equal areas, so every area is square. The areas are imaginary: do not draw them.\n", columns/d, rows/d, rows, columns)
	b.WriteString("- Use one uniform flat pure white background (#FFFFFF) everywhere: no scenery, ground, sky, floor, shadow, gradient or vignette.\n")
	b.WriteString("- Centre each sticker in its own area and keep it at most about 70% of that area's width and height, leaving even empty white margin on every side. Stickers never touch, overlap or cross an area boundary, and nothing touches the sheet edge.\n")
	b.WriteString("- Draw each sticker as one connected shape with a clear closed contour, thick even lines and flat colours. No gradients, glow, texture or soft shadows. A prop or small context element must stay attached to the sticker's subject.\n")
	fmt.Fprintf(&b, "- Exactly %d stickers: do not add, merge, duplicate or leave out any.\n", total)
	if hasLabels {
		b.WriteString("- Do not draw grid lines, frames, cell numbering, unrequested lettering, watermarks or signatures. Allow every requested label character, including digits.\n")
	} else {
		b.WriteString("- Do not draw grid lines, frames, numbers, text, captions, watermarks or signatures.\n")
	}
	return b.String()
}

func outlineLine(outlineWidth int, outlineColor string) string {
	if outlineWidth == 0 {
		return "Outline: none is added after cutting; keep the artwork edge solid and clean.\n"
	}
	return fmt.Sprintf("Outline: after cutting, each sticker gets a %d px %s outline at 512 px export plus a 4 px empty margin. Do not draw that outer outline or any white die-cut border yourself; the software adds it.\n",
		outlineWidth, outlineColor)
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

func (s *Service) loadFragments(ctx context.Context, ids []string) (map[string]Fragment, error) {
	found := make(map[string]Fragment, len(ids))
	if len(ids) == 0 {
		return found, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for index, id := range ids {
		args[index] = id
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+fragmentColumns+` FROM fragments f WHERE f.id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("studio: load fragments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanFragment(rows)
		if err != nil {
			return nil, fmt.Errorf("studio: scan fragment: %w", err)
		}
		found[item.ID] = item
	}
	return found, rows.Err()
}

func (s *Service) loadGroups(ctx context.Context, ids []string) (map[string]Group, error) {
	found := make(map[string]Group, len(ids))
	if len(ids) == 0 {
		return found, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for index, id := range ids {
		args[index] = id
	}
	rows, err := s.db.QueryContext(ctx, `SELECT `+groupColumns+` FROM fragment_groups WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("studio: load groups: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		item, err := scanGroup(rows)
		if err != nil {
			return nil, fmt.Errorf("studio: scan group: %w", err)
		}
		found[item.ID] = item
	}
	return found, rows.Err()
}

// groupMap loads the groups of the given fragments so selections can be ordered
// by group and carry the group name.
func (s *Service) groupMap(ctx context.Context, fragments map[string]Fragment) (map[string]Group, error) {
	ids := make([]string, 0, len(fragments))
	seen := map[string]bool{}
	for _, fragment := range fragments {
		if !seen[fragment.GroupID] {
			seen[fragment.GroupID] = true
			ids = append(ids, fragment.GroupID)
		}
	}
	return s.loadGroups(ctx, ids)
}

// selectionsFor resolves one ordered list of fragment ids into frozen
// selections, ordered by group then fragment name.
func selectionsFor(ids []string, fragments map[string]Fragment, groups map[string]Group) ([]Selection, error) {
	selections := make([]Selection, 0, len(ids))
	for _, id := range ids {
		fragment, ok := fragments[id]
		switch {
		case !ok:
			return nil, fmt.Errorf("%w: unknown fragment %q", ErrInvalid, id)
		case fragment.Archived:
			return nil, fmt.Errorf("%w: fragment %q is archived", ErrInvalid, id)
		}
		group := groups[fragment.GroupID]
		selections = append(selections, Selection{
			GroupID:    fragment.GroupID,
			GroupName:  group.Name,
			FragmentID: fragment.ID,
			Name:       fragment.Name,
			Prompt:     fragment.Prompt,
			Scope:      fragment.Scope,
		})
	}
	sort.SliceStable(selections, func(i, j int) bool {
		left, right := groups[selections[i].GroupID], groups[selections[j].GroupID]
		if left.SortOrder != right.SortOrder {
			return left.SortOrder < right.SortOrder
		}
		if left.Name != right.Name {
			return left.Name < right.Name
		}
		if selections[i].Name != selections[j].Name {
			return selections[i].Name < selections[j].Name
		}
		return selections[i].FragmentID < selections[j].FragmentID
	})
	return selections, nil
}

// candidates returns the selectable fragments of each group by group id.
func (s *Service) candidates(ctx context.Context, groupIDs []string, excludeIdentity bool) (map[string][]string, error) {
	pool := make(map[string][]string, len(groupIDs))
	if len(groupIDs) == 0 {
		return pool, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(groupIDs)), ",")
	args := make([]any, len(groupIDs))
	for index, id := range groupIDs {
		args[index] = id
	}
	scopeFilter := ""
	if excludeIdentity {
		scopeFilter = " AND scope != ?"
		args = append(args, ScopeIdentity)
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, group_id FROM fragments
		WHERE archived_at IS NULL AND group_id IN (`+placeholders+`)`+scopeFilter+` ORDER BY group_id, name, id`, args...)
	if err != nil {
		return nil, fmt.Errorf("studio: list candidates: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id, groupID string
		if err := rows.Scan(&id, &groupID); err != nil {
			return nil, fmt.Errorf("studio: scan candidate: %w", err)
		}
		pool[groupID] = append(pool[groupID], id)
	}
	return pool, rows.Err()
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func keys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for key := range set {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func finishSentence(text string) string {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasSuffix(text, ".") || strings.HasSuffix(text, "!") || strings.HasSuffix(text, "?") {
		return text
	}
	return text + "."
}
