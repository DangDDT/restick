package pack

import "errors"

const (
	StatusDraft           = "draft"
	StatusProcessing      = "processing"
	StatusPendingReview   = "pending_review"
	StatusAwaitingConfirm = "awaiting_confirm"
	StatusRejected        = "rejected"
	StatusPublished       = "published"
	StatusRevoked         = "revoked"
	SourceSheet           = "sheet"
	SourceFiles           = "files"

	RejectionCodeCopyright  = "copyright"
	RejectionCodeQuality    = "quality"
	RejectionCodeDuplicate  = "duplicate"
	RejectionCodeMisleading = "misleading"
	RejectionCodeExplicit   = "explicit"

	RevocationCodeDMCA          = "dmca"
	RevocationCodeMalicious     = "malicious"
	RevocationCodeAuthorRequest = "author_request"
	RevocationCodeTosBreach     = "tos_breach"
)

var (
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrForbidden         = errors.New("forbidden")
	ErrConflict          = errors.New("conflict")
	ErrNotFound          = errors.New("not found")
	ErrRevoked           = errors.New("pack_revoked")
)

type Pack struct {
	ID             string
	SubmitterID    string
	Slug           string
	Name           string
	Author         string
	Description    string
	Tags           []string
	CurrentVersion int
	ChecksumSHA256 string
	ThumbnailKey   string
	BundleKey      string
	StickerCount   int
	IPPolicy       string
	Status         string
	RejectReason   string
	SourceKind     string
	SourcePrefix   string
}

type Sticker struct {
	Filename   string
	EmojiHint  string
	Keywords   []string
	Width      int
	Height     int
	ObjectKey  string
	ThumbKey   string
	SortOrder  int
}

type Reason struct {
	Code string
	Text string
}

func DefaultRejectionReasons() []Reason {
	return []Reason{
		{Code: RejectionCodeCopyright, Text: "Copyright & Trademark Violation"},
		{Code: RejectionCodeQuality, Text: "Technical & Rendering Quality Issues"},
		{Code: RejectionCodeDuplicate, Text: "Duplicate / Spam Content"},
		{Code: RejectionCodeMisleading, Text: "Misleading Metadata / Impersonation"},
		{Code: RejectionCodeExplicit, Text: "Community Safety Violation"},
	}
}

func DefaultRevocationReasons() []Reason {
	return []Reason{
		{Code: RevocationCodeDMCA, Text: "DMCA Copyright Take-down Notice"},
		{Code: RevocationCodeMalicious, Text: "Reported Malicious / Abuse Content"},
		{Code: RevocationCodeAuthorRequest, Text: "Author Self-Withdrawal Request"},
		{Code: RevocationCodeTosBreach, Text: "Terms of Service Violation"},
	}
}

func AfterProcessStatus(submitterIsAdmin bool) string {
	if submitterIsAdmin {
		return StatusAwaitingConfirm
	}
	return StatusPendingReview
}

func CanSubmit(status string) bool {
	return status == StatusDraft || status == StatusRejected
}

func CanConfirm(p Pack, actorID string, actorIsAdmin bool) error {
	if !actorIsAdmin || p.SubmitterID != actorID {
		return ErrForbidden
	}
	if p.Status != StatusAwaitingConfirm {
		return ErrConflict
	}
	return nil
}

func CanApprove(p Pack, actorID string, actorIsAdmin bool) error {
	if !actorIsAdmin {
		return ErrForbidden
	}
	if p.SubmitterID == actorID {
		return ErrForbidden
	}
	if p.Status != StatusPendingReview {
		return ErrConflict
	}
	return nil
}

func CanRevoke(status string) error {
	if status != StatusPublished && status != StatusRevoked {
		return ErrConflict
	}
	return nil
}

func CanOwnerDeleteRejected(p Pack, actorID string) error {
	if p.SubmitterID != actorID {
		return ErrForbidden
	}
	if p.Status != StatusRejected {
		return ErrConflict
	}
	return nil
}

func CanOwnerRevokePublished(p Pack, actorID string) error {
	if p.SubmitterID != actorID {
		return ErrForbidden
	}
	if p.Status != StatusPublished {
		return ErrConflict
	}
	return nil
}

func CanEditPack(p Pack, actorID string, actorIsAdmin bool) error {
	if actorIsAdmin {
		return nil
	}
	if p.SubmitterID != actorID {
		return ErrForbidden
	}
	return nil
}

func CanEditSlug(status string) bool {
	return status == StatusDraft || status == StatusPendingReview || status == StatusAwaitingConfirm || status == StatusRejected
}

