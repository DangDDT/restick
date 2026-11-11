package httpx

import (
	"errors"
	"io"
	"net/http"

	"github.com/eykh/restick/api_svc/internal/assets"
	"github.com/eykh/restick/api_svc/internal/studio"
)

// admin_sheets.go serves the externally generated sticker sheets of a design.

func (a *admin) sheetImageURL(designID, sheetID string) string {
	return a.baseURL + "/api/v1/admin/designs/" + designID + "/sheets/" + sheetID + "/image"
}

func (a *admin) listSheets(w http.ResponseWriter, r *http.Request) {
	items, err := a.studio.ListSheets(r.Context(), r.PathValue("id"))
	if err != nil {
		writeStudioError(w, err)
		return
	}
	publicJSON(w, http.StatusOK, map[string]any{"items": items})
}

// uploadSheet answers 201 for a new sheet and 200 when the same bytes were
// already stored for the design (SaveSheet returns the existing row).
func (a *admin) uploadSheet(w http.ResponseWriter, r *http.Request) {
	designID := r.PathValue("id")
	a.upload(w, r, http.StatusCreated, func(reader io.Reader, filename string, fields map[string]string) (any, int, error) {
		// shortcut: a concurrent upload of the same bytes can be reported as 201, harmless for an idempotent upload.
		before, err := a.studio.ListSheets(r.Context(), designID)
		if err != nil {
			return nil, 0, err
		}
		sheet, err := a.studio.SaveSheet(r.Context(), designID, reader, filename, fields["provider"], fields["note"])
		if err != nil {
			return nil, 0, err
		}
		for _, existing := range before {
			if existing.ID == sheet.ID {
				return sheet, http.StatusOK, nil
			}
		}
		return sheet, 0, nil
	}, writeSheetUploadError)
}

// writeSheetUploadError keeps design and sheet domain errors (404, 409, 422 on
// bad provider) distinct from image validation errors.
func writeSheetUploadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, studio.ErrNotFound), errors.Is(err, studio.ErrInvalid),
		errors.Is(err, studio.ErrSheetLimit):
		writeStudioError(w, err)
	default:
		writeUploadError(w, err)
	}
}

func (a *admin) getSheetImage(w http.ResponseWriter, r *http.Request) {
	sheet, file, err := a.studio.OpenSheet(r.Context(), r.PathValue("id"), r.PathValue("sheetId"))
	if err != nil {
		switch {
		case errors.Is(err, studio.ErrNotFound), errors.Is(err, assets.ErrNotFound):
			publicError(w, http.StatusNotFound, "not_found", "sheet not found")
		default:
			publicError(w, http.StatusInternalServerError, "internal", "could not open sheet")
		}
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		publicError(w, http.StatusInternalServerError, "internal", "could not open sheet")
		return
	}
	w.Header().Set("Content-Type", sheet.MediaType)
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, sheet.Filename, info.ModTime(), file)
}

func (a *admin) deleteSheet(w http.ResponseWriter, r *http.Request) {
	if err := a.studio.DeleteSheet(r.Context(), r.PathValue("id"), r.PathValue("sheetId")); err != nil {
		writeStudioError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}
