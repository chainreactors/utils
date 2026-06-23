package ahocorasick

// OptimizedNFA represents an optimized Aho-Corasick automaton.
// Key optimizations:
// 1. Sparse construction-time transitions to keep build memory bounded
// 2. Precomputed root transitions: no failure link following for root
// 3. ByteClasses: reduced alphabet size
type OptimizedNFA struct {
	states       []optState
	byteClasses  *ByteClasses
	alphabetLen  int
	startState   StateID
	matchKind    MatchKind
	patternCount int
}

type optState struct {
	trans   []optTransition
	fail    StateID
	matches []PatternID
	depth   int
}

type optTransition struct {
	class uint16
	next  StateID
}

func buildOptimizedNFA(patterns [][]byte, bc *ByteClasses, matchKind MatchKind) *OptimizedNFA {
	alphabetLen := bc.NumClasses()

	nfa := &OptimizedNFA{
		byteClasses:  bc,
		alphabetLen:  alphabetLen,
		startState:   0,
		matchKind:    matchKind,
		patternCount: len(patterns),
	}

	nfa.buildTrie(patterns)
	nfa.buildFailureLinks()

	if matchKind == LeftmostFirst || matchKind == LeftmostLongest {
		nfa.propagateMatches()
	}

	nfa.precomputeRootTransitions()

	return nfa
}

func (nfa *OptimizedNFA) buildTrie(patterns [][]byte) {
	nfa.states = append(nfa.states, optState{
		fail:  0,
		depth: 0,
	})

	for patternID, pattern := range patterns {
		nfa.addPattern(pattern, PatternID(patternID))
	}
}

func (nfa *OptimizedNFA) addPattern(pattern []byte, patternID PatternID) {
	state := nfa.startState

	for _, b := range pattern {
		class := nfa.byteClasses.Get(b)

		if next := nfa.transition(state, class); next != 0 {
			state = next
		} else {
			newState := StateID(len(nfa.states))
			nfa.states = append(nfa.states, optState{
				fail:  0,
				depth: nfa.states[state].depth + 1,
			})
			nfa.setTransition(state, class, newState)
			state = newState
		}
	}

	nfa.states[state].matches = append(nfa.states[state].matches, patternID)
}

func (nfa *OptimizedNFA) buildFailureLinks() {
	queue := make([]StateID, 0, len(nfa.states))

	root := &nfa.states[nfa.startState]
	for _, edge := range root.trans {
		child := edge.next
		nfa.states[child].fail = nfa.startState
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		state := queue[0]
		queue = queue[1:]

		for _, edge := range nfa.states[state].trans {
			class := int(edge.class)
			child := edge.next
			queue = append(queue, child)

			fail := nfa.states[state].fail
			for {
				if next := nfa.transition(fail, class); next != 0 {
					nfa.states[child].fail = next
					break
				}
				if fail == nfa.startState {
					nfa.states[child].fail = nfa.startState
					break
				}
				fail = nfa.states[fail].fail
			}
		}
	}
}

func (nfa *OptimizedNFA) propagateMatches() {
	queue := make([]StateID, 0, len(nfa.states))

	root := &nfa.states[nfa.startState]
	for _, edge := range root.trans {
		queue = append(queue, edge.next)
	}

	for len(queue) > 0 {
		stateID := queue[0]
		queue = queue[1:]

		state := &nfa.states[stateID]

		for _, edge := range state.trans {
			queue = append(queue, edge.next)
		}

		if state.fail != nfa.startState {
			failMatches := nfa.states[state.fail].matches
			if len(failMatches) > 0 {
				state.matches = append(state.matches, failMatches...)
			}
		}
	}
}

func (nfa *OptimizedNFA) precomputeRootTransitions() {
	// Root state trans[class] == 0 means "stay at root".
	// No additional work needed; the DFA builder resolves this.
}

func (nfa *OptimizedNFA) transition(state StateID, class int) StateID {
	for _, edge := range nfa.states[state].trans {
		if int(edge.class) == class {
			return edge.next
		}
	}
	return 0
}

func (nfa *OptimizedNFA) setTransition(state StateID, class int, next StateID) {
	transitions := &nfa.states[state].trans
	for i := range *transitions {
		if int((*transitions)[i].class) == class {
			(*transitions)[i].next = next
			return
		}
	}
	*transitions = append(*transitions, optTransition{
		class: uint16(class),
		next:  next,
	})
}
