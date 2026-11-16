package model

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxFolderNameLength = 255

// ValidateFolderName vraća očišćeno ime (bez razmaka na početku i kraju) ili grešku ako ime nije dozvoljeno.

func ValidateFolderName(name string) (string, error) {
	name = strings.TrimSpace(name)

	if name == "" {
		return "", invalidName("Folder name must not be empty")
	}
	if utf8.RuneCountInString(name) > MaxFolderNameLength {
		return "", invalidName("Folder name is too long")
	}
	if name == "." || name == ".." {
		return "", invalidName("Folder name is not allowed")
	}
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		return "", invalidName(`Folder name must not contain any of / \ : * ? " < > |`)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", invalidName("Folder name must not contain control characters")
		}
	}

	return name, nil
}
