package proxy

import (
	"sync"
	"time"
)

// RecordAddonConfig controls how RecordAddon captures and delivers flows.
type RecordAddonConfig struct {
	MaxBodySnip int    // max bytes per body snapshot; 0 = unlimited; default 4096
	TagHeader   string // request header to extract as a tag (e.g. "X-Scan-Tag"); empty = disabled
	TagKey      string // key stored in FlowRecord.Tags; default "scan"
	StripTag    bool   // remove TagHeader from the request before forwarding

	// OnRecord is called for every completed HTTP transaction.
	// The consumer decides how to store/process the record.
	OnRecord func(*FlowRecord)

	// Enrich is called after building the FlowRecord but before OnRecord.
	// Use it to attach domain-specific metadata (e.g. rule match results).
	Enrich func(record *FlowRecord, raw *Flow)
}

// RecordAddon passively captures HTTP flows, converts them to FlowRecords,
// and delivers them via the OnRecord callback.
type RecordAddon struct {
	BaseAddon
	config  RecordAddonConfig
	pending sync.Map // flow UUID string → startInfo
}

type startInfo struct {
	time time.Time
	tag  string
}

// NewRecordAddon creates a recording addon with the given configuration.
func NewRecordAddon(config RecordAddonConfig) *RecordAddon {
	if config.MaxBodySnip == 0 {
		config.MaxBodySnip = 4096
	}
	if config.TagKey == "" {
		config.TagKey = "scan"
	}
	return &RecordAddon{config: config}
}

func (a *RecordAddon) Requestheaders(f *Flow) {
	info := startInfo{time: time.Now()}

	if a.config.TagHeader != "" && f.Request != nil {
		tag := f.Request.Header.Get(a.config.TagHeader)
		if tag != "" {
			info.tag = tag
			if a.config.StripTag {
				f.Request.Header.Del(a.config.TagHeader)
				if f.Request.raw != nil {
					f.Request.raw.Header.Del(a.config.TagHeader)
				}
			}
		}
	}

	a.pending.Store(f.Id.String(), info)
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
	var tag string
	if v, ok := a.pending.LoadAndDelete(f.Id.String()); ok {
		si := v.(startInfo)
		dur = time.Since(si.time)
		tag = si.tag
	}

	r := NewFlowRecord(f, a.config.MaxBodySnip)
	r.Duration = dur
	r.Error = errMsg

	if tag != "" {
		r.Tags[a.config.TagKey] = tag
	}

	if a.config.Enrich != nil {
		a.config.Enrich(r, f)
	}

	if a.config.OnRecord != nil {
		a.config.OnRecord(r)
	}
}

// TagHeader returns the configured tag header name.
func (a *RecordAddon) TagHeader() string {
	return a.config.TagHeader
}

// TagKey returns the configured tag key for FlowRecord.Tags.
func (a *RecordAddon) TagKey() string {
	return a.config.TagKey
}
