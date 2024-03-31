package slc

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
