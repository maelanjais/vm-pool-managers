package utils

import "unicode"

func FirstLetters(s string) string {
	for i, r := range s {
		if !unicode.IsLetter(r) {
			return s[:i] // retourne la sous-chaîne jusqu'à ce caractère
		}
	}
	return s // si tout est alphabétique
}
