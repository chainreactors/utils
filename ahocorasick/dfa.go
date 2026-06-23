package ahocorasick

const matchFlag uint32 = 1 << 31
const matchMask uint32 = matchFlag - 1

// DFA represents a fully compiled deterministic finite automaton.
type DFA struct {
	trans         []uint32
	matchIndex    []uint32
	matchData     []PatternID
	matchOverflow map[uint32][]PatternID
	byteClasses   *ByteClasses
	alphabetLen   int
	stride        int
	stride2       uint
	stateCount    int
	patternLens   []int
	matchKind     MatchKind
	startID       uint32
	startBytes    []byte
	patternBytes  [4]uint64
}

func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	return n + 1
}

func log2(n int) uint {
	var r uint
	for n >>= 1; n > 0; n >>= 1 {
		r++
	}
	return r
}

func buildDFA(nfa *OptimizedNFA, patterns [][]byte, matchKind MatchKind) *DFA {
	numStates := len(nfa.states)
	alphabetLen := nfa.alphabetLen
	stride := nextPow2(alphabetLen)
	stride2 := log2(stride)

	d := &DFA{
		byteClasses: nfa.byteClasses,
		alphabetLen: alphabetLen,
		stride:      stride,
		stride2:     stride2,
		stateCount:  numStates,
		matchKind:   matchKind,
		startID:     uint32(nfa.startState) << stride2,
	}

	d.patternLens = make([]int, len(patterns))
	startByteSet := [256]bool{}
	for i, p := range patterns {
		d.patternLens[i] = len(p)
		if len(p) > 0 {
			startByteSet[p[0]] = true
		}
		for _, b := range p {
			d.patternBytes[b/64] |= 1 << (b % 64)
		}
	}

	for b := 0; b < 256; b++ {
		if startByteSet[b] {
			d.startBytes = append(d.startBytes, byte(b))
		}
	}

	tableSize := numStates * stride
	d.trans = make([]uint32, tableSize)

	encodeTransition := func(next StateID) uint32 {
		premultiplied := uint32(next) << stride2
		if len(nfa.states[next].matches) > 0 {
			premultiplied |= matchFlag
		}
		return premultiplied
	}

	root := nfa.startState
	rootOffset := int(root) * stride
	rootTransition := encodeTransition(root)
	for class := 0; class < alphabetLen; class++ {
		d.trans[rootOffset+class] = rootTransition
	}

	queue := make([]StateID, 0, numStates)
	for _, edge := range nfa.states[root].trans {
		d.trans[rootOffset+int(edge.class)] = encodeTransition(edge.next)
		queue = append(queue, edge.next)
	}

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]

		rowOffset := int(state) * stride
		failOffset := int(nfa.states[state].fail) * stride
		copy(d.trans[rowOffset:rowOffset+alphabetLen], d.trans[failOffset:failOffset+alphabetLen])

		for _, edge := range nfa.states[state].trans {
			d.trans[rowOffset+int(edge.class)] = encodeTransition(edge.next)
			queue = append(queue, edge.next)
		}
	}

	var totalMatches int
	var overflowStates int
	for si := 0; si < numStates; si++ {
		count := len(nfa.states[si].matches)
		if count == 0 {
			continue
		}
		if totalMatches <= 0xFFFF && count <= 0xFFFF {
			totalMatches += count
		} else {
			overflowStates++
		}
	}

	d.matchData = make([]PatternID, 0, totalMatches)
	d.matchIndex = make([]uint32, numStates)
	if overflowStates > 0 {
		d.matchOverflow = make(map[uint32][]PatternID, overflowStates)
	}

	for si := 0; si < numStates; si++ {
		matches := nfa.states[si].matches
		if len(matches) == 0 {
			continue
		}

		offset := len(d.matchData)
		count := len(matches)

		if offset <= 0xFFFF && count <= 0xFFFF {
			d.matchData = append(d.matchData, matches...)
			d.matchIndex[si] = uint32(offset<<16) | uint32(count)
		} else {
			d.matchIndex[si] = 0xFFFFFFFF
			d.matchOverflow[uint32(si)] = matches
		}
	}

	return d
}

func (d *DFA) getMatches(sid uint32) []PatternID {
	idx := sid >> d.stride2
	packed := d.matchIndex[idx]
	if packed == 0 {
		return nil
	}
	if packed == 0xFFFFFFFF {
		return d.matchOverflow[idx]
	}
	offset := int(packed >> 16)
	count := int(packed & 0xFFFF)
	return d.matchData[offset : offset+count]
}

// MemoryUsage returns the approximate heap memory used by this DFA in bytes.
func (d *DFA) MemoryUsage() int {
	return len(d.trans)*4 +
		len(d.matchIndex)*4 +
		len(d.matchData)*4 + len(d.patternLens)*8
}
