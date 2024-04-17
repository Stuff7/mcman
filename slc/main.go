package slc

import "strings"

func Get[S any](slice []S, idx int) *S {
	if idx < 0 || idx >= len(slice) {
		return nil
	}
	return &slice[idx]
}

func Flatten[S any](slice [][]S) []S {
	flat := []S{}
	for _, t := range slice {
		flat = append(flat, t...)
	}
	return flat
}

func Filter[S any](s []S, f func(S) bool) []S {
	var sf []S
	for _, s := range s {
		if f(s) {
			sf = append(sf, s)
		}
	}
	return sf
}

func Map[S, M any](ts []S, f func(S) M) []M {
	us := make([]M, len(ts))
	for i := range ts {
		us[i] = f(ts[i])
	}
	return us
}

func Last[S any](slice []S) *S {
	if l := len(slice); l != 0 {
		return &slice[l-1]
	}
	return nil
}

func LevenshteinDistance(s1, s2 string) int {
	len1 := len(s1)
	len2 := len(s2)

	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}

	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	for i, char1 := range s1 {
		for j, char2 := range s2 {
			cost := 0
			if char1 != char2 {
				cost = 1
			}
			matrix[i+1][j+1] = min(min(matrix[i][j+1]+1, matrix[i+1][j]+1), matrix[i][j]+cost)
		}
	}

	return matrix[len1][len2]
}

func FuzzyStringCompare(s1, s2 string) float32 {
	length := float32(max(len(s1), len(s2)))
	return float32(LevenshteinDistance(strings.ToLower(s1), strings.ToLower(s2))) / length
}
