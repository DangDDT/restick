package objectkey

import (
	"errors"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrInvalidKey = errors.New("invalid object key")
	packPattern   = regexp.MustCompile(`^packs/[0-9a-fA-F-]{36}/v[0-9]+/[A-Za-z0-9._-]+$`)
	uploadPattern = regexp.MustCompile(`^uploads/[0-9a-fA-F-]{36}/[0-9a-fA-F-]{36}/[A-Za-z0-9._-]+$`)
)

func ValidateProcessed(key string) error {
	if !packPattern.MatchString(key) {
		return ErrInvalidKey
	}
	return rejectDotDot(key)
}

func ValidateUpload(key string, userID string) error {
	if !uploadPattern.MatchString(key) {
		return ErrInvalidKey
	}
	if err := rejectDotDot(key); err != nil {
		return err
	}
	prefix := "uploads/" + strings.ToLower(userID) + "/"
	if !strings.HasPrefix(strings.ToLower(key), strings.ToLower(prefix)) {
		return ErrInvalidKey
	}
	return nil
}

func BundleKey(packID string, version int) string {
	return path.Join("packs", packID, "v"+strconv.Itoa(version), "bundle.zip")
}

func rejectDotDot(key string) error {
	if strings.Contains(key, "..") || strings.HasPrefix(key, "/") {
		return ErrInvalidKey
	}
	return nil
}
