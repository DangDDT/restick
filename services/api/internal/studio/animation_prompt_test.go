package studio

import (
	"context"
	"strings"
	"testing"
)

// TestCompileAnimationPromptCarriesSubjectMotionAndRules pins the two facts the
// operator relies on: the design's character description is the subject and the
// requested motion is in the prompt. Wording itself stays free to change.
func TestCompileAnimationPromptCarriesSubjectMotionAndRules(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()
	asset := mustReferenceAsset(t, service, 0x02)
	character, err := service.CreateCharacter(ctx, CharacterDraft{
		Name:              "Mèo",
		Description:       "a round orange cat with a red scarf",
		ReferenceAssetIDs: []string{asset.ID},
	})
	if err != nil {
		t.Fatalf("CreateCharacter: %v", err)
	}
	draft := draftWithReference(t, sheetDraft(nil, nil), asset.ID)
	draft.CharacterID = character.ID
	design, err := service.SaveDesign(ctx, draft)
	if err != nil {
		t.Fatalf("SaveDesign: %v", err)
	}

	prompt, err := service.CompileAnimation(ctx, design.ID, "waving one paw")
	if err != nil {
		t.Fatalf("CompileAnimation: %v", err)
	}
	if !strings.Contains(prompt, "a round orange cat with a red scarf") || !strings.Contains(prompt, "waving one paw") {
		t.Fatalf("prompt misses subject or motion:\n%s", prompt)
	}
	if !strings.Contains(strings.ToLower(prompt), "white") || !strings.Contains(prompt, "3 second") {
		t.Fatalf("prompt misses the background or loop rule:\n%s", prompt)
	}
}

// TestCompileAnimationFallsBackWithoutDesign proves a pack whose design is gone
// still compiles a usable prompt.
func TestCompileAnimationFallsBackWithoutDesign(t *testing.T) {
	service, _ := newTestService(t)
	prompt, err := service.CompileAnimation(context.Background(), "", "blinking twice")
	if err != nil {
		t.Fatalf("CompileAnimation: %v", err)
	}
	if !strings.Contains(prompt, "blinking twice") || !strings.Contains(prompt, "sticker character") {
		t.Fatalf("fallback prompt = %q", prompt)
	}
}
