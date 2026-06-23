package ahocorasick

import "bytes"

// Automaton is the compiled Aho-Corasick multi-pattern matcher.
type Automaton struct {
	dfa       *DFA
	patterns  [][]byte
	matchKind MatchKind
}

// Find returns the first match in haystack starting at or after position start.
// Returns nil if no match is found.
func (a *Automaton) Find(haystack []byte, start int) *Match {
	m, ok := a.FindMatch(haystack, start)
	if !ok {
		return nil
	}
	return &m
}

// FindMatch returns the first match in haystack starting at or after position start.
func (a *Automaton) FindMatch(haystack []byte, start int) (Match, bool) {
	if start >= len(haystack) {
		return Match{}, false
	}

	d := a.dfa

	sb := d.startBytes
	remaining := len(haystack) - start
	if len(sb) > 0 && remaining >= 128 {
		skip := findEarliestStartByte(haystack[start:], sb)
		if skip < 0 {
			return Match{}, false
		}
		start += skip
	}

	trans := d.trans
	classes := &d.byteClasses.classes
	sid := d.startID
	patternLens := d.patternLens

	_ = trans[len(trans)-1]

	var bestMatch Match
	found := false

	for i := start; i < len(haystack); i++ {
		raw := trans[int(sid)+int(classes[haystack[i]])]

		if raw&matchFlag == 0 {
			sid = raw
			continue
		}

		sid = raw & matchMask
		matches := d.getMatches(sid)
		if len(matches) == 0 {
			continue
		}

		patternID := matches[0]
		matchEnd := i + 1
		matchStart := matchEnd - patternLens[patternID]

		m := Match{
			PatternID: int(patternID),
			Start:     matchStart,
			End:       matchEnd,
		}

		if a.matchKind == LeftmostFirst {
			return m, true
		}

		if !found || m.Len() > bestMatch.Len() {
			bestMatch = m
			found = true
		}
	}

	return bestMatch, found
}

// FindAt returns the first match starting exactly at position start.
// Returns nil if no match starts at the given position.
func (a *Automaton) FindAt(haystack []byte, start int) *Match {
	m, ok := a.FindAtMatch(haystack, start)
	if !ok {
		return nil
	}
	return &m
}

// FindAtMatch returns the first match starting exactly at position start.
func (a *Automaton) FindAtMatch(haystack []byte, start int) (Match, bool) {
	if start >= len(haystack) {
		return Match{}, false
	}

	d := a.dfa
	trans := d.trans
	classes := &d.byteClasses.classes
	sid := d.startID
	startID := d.startID
	patternLens := d.patternLens

	_ = trans[len(trans)-1]

	var bestMatch Match
	found := false

	for i := start; i < len(haystack); i++ {
		prevSid := sid
		raw := trans[int(sid)+int(classes[haystack[i]])]

		if prevSid == startID && i > start {
			break
		}

		if raw&matchFlag == 0 {
			sid = raw
			continue
		}

		sid = raw & matchMask
		for _, patternID := range d.getMatches(sid) {
			patLen := patternLens[patternID]
			matchEnd := i + 1
			matchStart := matchEnd - patLen

			if matchStart != start {
				continue
			}

			m := Match{
				PatternID: int(patternID),
				Start:     matchStart,
				End:       matchEnd,
			}

			if a.matchKind == LeftmostFirst {
				return m, true
			}

			if !found || m.Len() > bestMatch.Len() {
				bestMatch = m
				found = true
			}
		}
	}

	return bestMatch, found
}

// IsMatch returns true if any pattern matches anywhere in the haystack.
// Zero allocations, minimal branching.
func (a *Automaton) IsMatch(haystack []byte) bool {
	d := a.dfa

	sb := d.startBytes
	if len(sb) > 0 {
		start := findEarliestStartByte(haystack, sb)
		if start < 0 {
			return false
		}
		haystack = haystack[start:]
	}

	trans := d.trans
	classes := &d.byteClasses.classes
	var sid uint32 // startID is always 0

	if len(trans) > 0 {
		_ = trans[len(trans)-1]
	}

	for i := 0; i < len(haystack); i++ {
		raw := trans[int(sid)+int(classes[haystack[i]])]
		if raw&matchFlag != 0 {
			return true
		}
		sid = raw

		if sid == 0 && len(sb) > 0 && i+1 < len(haystack) {
			skip := findEarliestStartByte(haystack[i+1:], sb)
			if skip < 0 {
				return false
			}
			i += skip
		}
	}

	return false
}

func findEarliestStartByte(data []byte, startBytes []byte) int {
	earliest := -1
	for _, b := range startBytes {
		if idx := bytes.IndexByte(data, b); idx >= 0 {
			if earliest < 0 || idx < earliest {
				earliest = idx
			}
		}
	}
	return earliest
}

// FindAll returns all non-overlapping matches in the haystack.
// If n >= 0, at most n matches are returned.
func (a *Automaton) FindAll(haystack []byte, n int) []Match {
	if len(haystack) == 0 {
		return nil
	}

	var matches []Match
	a.forEachMatch(haystack, false, n, func(m Match) bool {
		matches = append(matches, m)
		return true
	})
	return matches
}

// FindAllOverlapping returns all overlapping matches in the haystack.
func (a *Automaton) FindAllOverlapping(haystack []byte) []Match {
	var matches []Match
	a.forEachMatch(haystack, true, -1, func(m Match) bool {
		matches = append(matches, m)
		return true
	})
	return matches
}

func (a *Automaton) forEachMatch(haystack []byte, overlapping bool, n int, emit func(Match) bool) {
	if overlapping {
		a.forEachOverlappingMatch(haystack, n, emit)
		return
	}
	a.forEachNonOverlappingMatch(haystack, n, emit)
}

func (a *Automaton) forEachNonOverlappingMatch(haystack []byte, n int, emit func(Match) bool) {
	if len(haystack) == 0 {
		return
	}

	d := a.dfa
	trans := d.trans
	classes := &d.byteClasses.classes
	patternLens := d.patternLens
	var sid uint32

	if len(trans) > 0 {
		_ = trans[len(trans)-1]
	}

	emitted := 0

	for i := 0; i < len(haystack); i++ {
		if n >= 0 && emitted >= n {
			break
		}

		raw := trans[int(sid)+int(classes[haystack[i]])]

		if raw&matchFlag == 0 {
			sid = raw
			continue
		}

		sid = raw & matchMask
		allMatches := d.getMatches(sid)
		if len(allMatches) == 0 {
			continue
		}

		patternID := allMatches[0]
		patLen := patternLens[patternID]
		matchEnd := i + 1
		matchStart := matchEnd - patLen

		emitted++
		if !emit(Match{
			PatternID: int(patternID),
			Start:     matchStart,
			End:       matchEnd,
		}) {
			return
		}

		if matchEnd > i+1 {
			i = matchEnd - 1
		}
		sid = 0
	}
}

func (a *Automaton) forEachOverlappingMatch(haystack []byte, n int, emit func(Match) bool) {
	d := a.dfa
	trans := d.trans
	classes := &d.byteClasses.classes
	sid := d.startID
	patternLens := d.patternLens

	if len(trans) > 0 {
		_ = trans[len(trans)-1]
	}

	emitted := 0
	for i, b := range haystack {
		if n >= 0 && emitted >= n {
			break
		}

		raw := trans[int(sid)+int(classes[b])]

		if raw&matchFlag == 0 {
			sid = raw
			continue
		}

		sid = raw & matchMask
		for _, patternID := range d.getMatches(sid) {
			if n >= 0 && emitted >= n {
				return
			}
			matchEnd := i + 1
			matchStart := matchEnd - patternLens[patternID]

			emitted++
			if !emit(Match{
				PatternID: int(patternID),
				Start:     matchStart,
				End:       matchEnd,
			}) {
				return
			}
		}
	}
}

// Count returns the number of non-overlapping matches in the haystack.
func (a *Automaton) Count(haystack []byte) int {
	count := 0
	pos := 0

	for pos < len(haystack) {
		m, ok := a.FindMatch(haystack, pos)
		if !ok {
			break
		}
		count++
		pos = m.End
		if pos <= m.Start {
			pos = m.Start + 1
		}
	}

	return count
}

// PatternCount returns the number of patterns in the automaton.
func (a *Automaton) PatternCount() int {
	return len(a.patterns)
}

// Pattern returns the pattern bytes at the given index.
func (a *Automaton) Pattern(id int) []byte {
	if id < 0 || id >= len(a.patterns) {
		return nil
	}
	return a.patterns[id]
}

// StateCount returns the number of states in the underlying automaton.
func (a *Automaton) StateCount() int {
	return a.dfa.stateCount
}

// MatchKindOf returns the match semantics used by this automaton.
func (a *Automaton) MatchKindOf() MatchKind {
	return a.matchKind
}
