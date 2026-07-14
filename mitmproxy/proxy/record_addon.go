package proxy

import (
	"sync"
	"time"
)

// RecordAddon passively captures HTTP flows, converts them to FlowRecords,
// and delivers them via the OnRecord callback. It handles only the timing
// measurement and Flow→FlowRecord conversion; any domain-specific logic
// (tag extraction, enrichment, storage) belongs in the consumer's callback.
type RecordAddon struct {
	BaseAddon

	MaxBodySnip int                   // max bytes per body snapshot; 0 = unlimited
	OnRecord    func(*FlowRecord)     // called for every completed HTTP transaction

	pending sync.Map // flow UUID string → time.Time
}

// NewRecordAddon creates a recording addon. OnRecord must not be nil.
func NewRecordAddon(onRecord func(*FlowRecord)) *RecordAddon {
	return &RecordAddon{OnRecord: onRecord}
}

func (a *RecordAddon) Requestheaders(f *Flow) {
	a.pending.Store(f.Id.String(), time.Now())
}

func (a *RecordAddon) Response(f *Flow) {
	a.record(f, "")
}

func (a *RecordAddon) RequestError(f *Flow, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	a.record(f, msg)
}

func (a *RecordAddon) record(f *Flow, errMsg string) {
	var dur time.Duration
	if v, ok := a.pending.LoadAndDelete(f.Id.String()); ok {
		dur = time.Since(v.(time.Time))
	}

	r := NewFlowRecord(f, a.MaxBodySnip)
	r.Duration = dur
	r.Error = errMsg

	if a.OnRecord != nil {
		a.OnRecord(r)
	}
}
