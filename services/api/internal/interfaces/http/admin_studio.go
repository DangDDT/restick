package httpx

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/studio"
)

// admin_studio.go serves the fragment library, character profiles, private
// reference images and designs.

func queryInt(r *http.Request, name string) int {
	value, _ := strconv.Atoi(r.URL.Query().Get(name))
	return value
}

func (a *admin) listGroups(w http.ResponseWriter, r *http.Request) {
	items, err := a.studio.ListGroups(r.Context())
	if err != nil {
		writeStudioError(w, err)
		return
	}
	if items == nil {
		items = []studio.Group{}
	}
	publicJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *admin) createGroup(w http.ResponseWriter, r *http.Request) {
	var draft studio.GroupDraft
	if !decodeJSON(w, r, &draft) {
		return
	}
	group, err := a.studio.CreateGroup(r.Context(), draft)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusCreated, group)
}

func (a *admin) updateGroup(w http.ResponseWriter, r *http.Request) {
	var draft studio.GroupDraft
	if !decodeJSON(w, r, &draft) {
		return
	}
	group, err := a.studio.UpdateGroup(r.Context(), r.PathValue("id"), draft)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, group)
}

func (a *admin) deleteGroup(w http.ResponseWriter, r *http.Request) {
	if err := a.studio.DeleteGroup(r.Context(), r.PathValue("id")); err != nil {
		writeStudioError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (a *admin) listFragments(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filter := studio.FragmentFilter{
		Query:           query.Get("q"),
		GroupID:         query.Get("group_id"),
		Scope:           studio.Scope(query.Get("scope")),
		IncludeArchived: query.Get("include_archived") == "true",
		Page:            queryInt(r, "page"),
		PageSize:        queryInt(r, "page_size"),
	}
	page, err := a.studio.ListFragments(r.Context(), filter)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, page)
}

func (a *admin) createFragment(w http.ResponseWriter, r *http.Request) {
	var draft studio.FragmentDraft
	if !decodeJSON(w, r, &draft) {
		return
	}
	fragment, err := a.studio.CreateFragment(r.Context(), draft)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusCreated, fragment)
}

func (a *admin) updateFragment(w http.ResponseWriter, r *http.Request) {
	var draft studio.FragmentDraft
	if !decodeJSON(w, r, &draft) {
		return
	}
	fragment, err := a.studio.UpdateFragment(r.Context(), r.PathValue("id"), draft)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, fragment)
}

func (a *admin) archiveFragment(w http.ResponseWriter, r *http.Request) {
	if _, err := a.studio.ArchiveFragment(r.Context(), r.PathValue("id")); err != nil {
		writeStudioError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (a *admin) listCharacters(w http.ResponseWriter, r *http.Request) {
	items, err := a.studio.ListCharacters(r.Context())
	if err != nil {
		writeStudioError(w, err)
		return
	}
	if items == nil {
		items = []studio.Character{}
	}
	publicJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *admin) getCharacter(w http.ResponseWriter, r *http.Request) {
	character, err := a.studio.GetCharacter(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, character)
}

func (a *admin) createCharacter(w http.ResponseWriter, r *http.Request) {
	var draft studio.CharacterDraft
	if !decodeJSON(w, r, &draft) {
		return
	}
	character, err := a.studio.CreateCharacter(r.Context(), draft)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusCreated, character)
}

func (a *admin) updateCharacter(w http.ResponseWriter, r *http.Request) {
	var draft studio.CharacterDraft
	if !decodeJSON(w, r, &draft) {
		return
	}
	character, err := a.studio.UpdateCharacter(r.Context(), r.PathValue("id"), draft)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, character)
}

func (a *admin) archiveCharacter(w http.ResponseWriter, r *http.Request) {
	if _, err := a.studio.ArchiveCharacter(r.Context(), r.PathValue("id")); err != nil {
		writeStudioError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func (a *admin) uploadReferenceAsset(w http.ResponseWriter, r *http.Request) {
	a.upload(w, r, http.StatusCreated, func(reader io.Reader, filename string) (any, error) {
		return a.studio.SaveReferenceAsset(r.Context(), reader, filename)
	}, writeUploadError)
}

// getReferenceAsset serves the private preview. It is the only route that
// returns reference bytes; the public object routes never list them.
func (a *admin) getReferenceAsset(w http.ResponseWriter, r *http.Request) {
	asset, file, err := a.studio.OpenReferenceAsset(r.Context(), r.PathValue("id"))
	if err != nil {
		switch {
		case errors.Is(err, studio.ErrNotFound), errors.Is(err, assets.ErrNotFound):
			publicError(w, http.StatusNotFound, "not_found", "reference asset not found")
		default:
			publicError(w, http.StatusInternalServerError, "internal", "could not open reference asset")
		}
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not open reference asset")
		return
	}
	w.Header().Set("Content-Type", asset.ContentType)
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, asset.Filename, info.ModTime(), file)
}

func (a *admin) listDesigns(w http.ResponseWriter, r *http.Request) {
	page, err := a.studio.ListDesigns(r.Context(), studio.DesignFilter{
		Query:    r.URL.Query().Get("q"),
		Page:     queryInt(r, "page"),
		PageSize: queryInt(r, "page_size"),
	})
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, page)
}

func (a *admin) getDesign(w http.ResponseWriter, r *http.Request) {
	design, err := a.studio.GetDesign(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, design)
}

// saveDesign always creates: a body id would silently rewrite an existing row.
func (a *admin) saveDesign(w http.ResponseWriter, r *http.Request) {
	draft := studio.Draft{OutlineWidth: studio.DefaultOutlineWidth}
	if !decodeJSON(w, r, &draft) {
		return
	}
	draft.ID = ""
	design, err := a.studio.SaveDesign(r.Context(), draft)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusCreated, design)
}

func (a *admin) updateDesign(w http.ResponseWriter, r *http.Request) {
	draft := studio.Draft{OutlineWidth: studio.DefaultOutlineWidth}
	if !decodeJSON(w, r, &draft) {
		return
	}
	draft.ID = r.PathValue("id")
	design, err := a.studio.SaveDesign(r.Context(), draft)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, design)
}

func (a *admin) deleteDesign(w http.ResponseWriter, r *http.Request) {
	if err := a.studio.DeleteDesign(r.Context(), r.PathValue("id")); err != nil {
		writeStudioError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

type compiledDesign struct {
	CompiledPrompt string          `json:"compiled_prompt"`
	Snapshot       studio.Snapshot `json:"snapshot"`
}

func (a *admin) compileDesign(w http.ResponseWriter, r *http.Request) {
	draft := studio.Draft{OutlineWidth: studio.DefaultOutlineWidth}
	if !decodeJSON(w, r, &draft) {
		return
	}
	compiled, err := a.studio.Compile(r.Context(), draft)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, compiledDesign{CompiledPrompt: compiled.Text, Snapshot: compiled.Snapshot})
}

// suggestDesign never saves; the operator gets a draft back and decides.
func (a *admin) suggestDesign(w http.ResponseWriter, r *http.Request) {
	draft := studio.Draft{OutlineWidth: studio.DefaultOutlineWidth}
	if !decodeJSON(w, r, &draft) {
		return
	}
	suggested, err := a.studio.Suggest(r.Context(), draft, nil)
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, suggested)
}
