package format

import (
	"bytes"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"

	"github.com/francoispqt/gojay"
	"github.com/wiggin77/logr"
)

// JSON formats log records as JSON.
type JSON struct {
	// DisableTimestamp disables output of timestamp field.
	DisableTimestamp bool
	// DisableLevel disables output of level field.
	DisableLevel bool
	// DisableMsg disables output of msg field.
	DisableMsg bool
	// DisableContext disables output of all context fields.
	DisableContext bool
	// DisableStacktrace disables output of stack trace.
	DisableStacktrace bool

	// TimestampFormat is an optional format for timestamps. If empty
	// then DefTimestampFormat is used.
	TimestampFormat string

	// Deprecated: this has no effect.
	Indent string

	// EscapeHTML determines if certain characters (e.g. `<`, `>`, `&`)
	// are escaped.
	EscapeHTML bool

	// KeyTimestamp overrides the timestamp field key name.
	KeyTimestamp string

	// KeyLevel overrides the level field key name.
	KeyLevel string

	// KeyMsg overrides the msg field key name.
	KeyMsg string

	// KeyContextFields when not empty will group all context fields
	// under this key.
	KeyContextFields string

	// KeyStacktrace overrides the stacktrace field key name.
	KeyStacktrace string

	// ContextSorter allows custom sorting for the context fields.
	// A new slice must be returned and the original unmodified.
	ContextSorter func(fields []logr.Field) []logr.Field

	once sync.Once
}

// Format converts a log record to bytes in JSON format.
func (j *JSON) Format(rec *logr.LogRec, stacktrace bool, buf *bytes.Buffer) (*bytes.Buffer, error) {
	j.once.Do(j.applyDefaultKeyNames)

	if buf == nil {
		buf = &bytes.Buffer{}
	}
	enc := gojay.BorrowEncoder(buf)
	defer func() {
		enc.Release()
	}()

	sorter := j.ContextSorter
	if sorter == nil {
		sorter = j.defaultContextSorter
	}

	jlr := JSONLogRec{
		LogRec:     rec,
		JSON:       j,
		stacktrace: stacktrace,
		sorter:     sorter,
	}

	err := enc.EncodeObject(jlr)
	if err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf, nil
}

func (j *JSON) applyDefaultKeyNames() {
	if j.KeyTimestamp == "" {
		j.KeyTimestamp = "timestamp"
	}
	if j.KeyLevel == "" {
		j.KeyLevel = "level"
	}
	if j.KeyMsg == "" {
		j.KeyMsg = "msg"
	}
	if j.KeyStacktrace == "" {
		j.KeyStacktrace = "stacktrace"
	}
}

// defaultContextSorter sorts the context fields alphabetically by key.
// A new slice must be returned and the original unmodified.
func (j *JSON) defaultContextSorter(fields []logr.Field) []logr.Field {
	sorted := make([]logr.Field, len(fields))
	copy(sorted, fields)

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Key < sorted[j].Key
	})
	return sorted
}

// JSONLogRec decorates a LogRec adding JSON encoding.
type JSONLogRec struct {
	*logr.LogRec
	*JSON
	stacktrace bool
	sorter     func(fields []logr.Field) []logr.Field
}

// MarshalJSONObject encodes the LogRec as JSON.
func (rec JSONLogRec) MarshalJSONObject(enc *gojay.Encoder) {
	if !rec.DisableTimestamp {
		timestampFmt := rec.TimestampFormat
		if timestampFmt == "" {
			timestampFmt = logr.DefTimestampFormat
		}
		time := rec.Time()
		enc.AddTimeKey(rec.KeyTimestamp, &time, timestampFmt)
	}
	if !rec.DisableLevel {
		enc.AddStringKey(rec.KeyLevel, rec.Level().Name)
	}
	if !rec.DisableMsg {
		enc.AddStringKey(rec.KeyMsg, rec.Msg())
	}
	if !rec.DisableContext {
		ctxFields := rec.sorter(rec.Fields())
		if rec.KeyContextFields != "" {
			enc.AddObjectKey(rec.KeyContextFields, jsonFields(ctxFields))
		} else {
			if len(ctxFields) > 0 {
				for _, cf := range ctxFields {
					key := rec.prefixCollision(cf.Key)
					encodeField(enc, key, cf)
				}
			}
		}
	}
	if rec.stacktrace && !rec.DisableStacktrace {
		frames := rec.StackFrames()
		if len(frames) > 0 {
			enc.AddArrayKey(rec.KeyStacktrace, stackFrames(frames))
		}
	}

}

// IsNil returns true if the LogRec pointer is nil.
func (rec JSONLogRec) IsNil() bool {
	return rec.LogRec == nil
}

func (rec JSONLogRec) prefixCollision(key string) string {
	switch key {
	case rec.KeyTimestamp, rec.KeyLevel, rec.KeyMsg, rec.KeyStacktrace:
		return rec.prefixCollision("_" + key)
	}
	return key
}

type stackFrames []runtime.Frame

// MarshalJSONArray encodes stackFrames slice as JSON.
func (s stackFrames) MarshalJSONArray(enc *gojay.Encoder) {
	for _, frame := range s {
		enc.AddObject(stackFrame(frame))
	}
}

// IsNil returns true if stackFrames is empty slice.
func (s stackFrames) IsNil() bool {
	return len(s) == 0
}

type stackFrame runtime.Frame

// MarshalJSONArray encodes stackFrame as JSON.
func (f stackFrame) MarshalJSONObject(enc *gojay.Encoder) {
	enc.AddStringKey("Function", f.Function)
	enc.AddStringKey("File", f.File)
	enc.AddIntKey("Line", f.Line)
}

func (f stackFrame) IsNil() bool {
	return false
}

type jsonFields []logr.Field

// MarshalJSONObject encodes Fields to JSON.
func (f jsonFields) MarshalJSONObject(enc *gojay.Encoder) {
	for _, ctxField := range f {
		encodeField(enc, ctxField.Key, ctxField)
	}
}

// IsNil returns true if fields array is nil.
func (f jsonFields) IsNil() bool {
	return f == nil
}

func encodeField(enc *gojay.Encoder, key string, field logr.Field) {
	switch field.Type {
	case logr.UnknownType:
		enc.AddStringKey(field.Key, "UnknownType")
	case logr.ArrayMarshalerType:
		enc.
	case logr.ObjectMarshalerType:
	case logr.BinaryType:
	case logr.BoolType:
		enc.AddBoolKey(key, bool(field.Integer))
	case logr.ByteStringType:
	case logr.Complex128Type:
	case logr.Complex64Type:
	case logr.DurationType:
	case logr.Float64Type:
	case logr.Float32Type:
	case logr.Int64Type:
	case logr.Int32Type:
	case logr.Int16Type:
	case logr.Int8Type:
	case logr.StringType:
	case logr.TimeType:
	case logr.TimeFullType:
	case logr.Uint64Type:
	case logr.Uint32Type:
	case logr.Uint16Type:
	case logr.Uint8Type:
	case logr.UintptrType:
	case logr.ReflectType:
	case logr.NamespaceType:
	case logr.StringerType:
	case logr.ErrorType:
	case logr.SkipType:

	case gojay.MarshalerJSONObject:
		enc.AddObjectKey(key, vt)
	case gojay.MarshalerJSONArray:
		enc.AddArrayKey(key, vt)
	case string:
		enc.AddStringKey(key, vt)
	case error:
		enc.AddStringKey(key, vt.Error())
	case bool:
		enc.AddBoolKey(key, vt)
	case int:
		enc.AddIntKey(key, vt)
	case int64:
		enc.AddInt64Key(key, vt)
	case int32:
		enc.AddIntKey(key, int(vt))
	case int16:
		enc.AddIntKey(key, int(vt))
	case int8:
		enc.AddIntKey(key, int(vt))
	case uint64:
		enc.AddIntKey(key, int(vt))
	case uint32:
		enc.AddIntKey(key, int(vt))
	case uint16:
		enc.AddIntKey(key, int(vt))
	case uint8:
		enc.AddIntKey(key, int(vt))
	case float64:
		enc.AddFloatKey(key, vt)
	case float32:
		enc.AddFloat32Key(key, vt)
	case *gojay.EmbeddedJSON:
		enc.AddEmbeddedJSONKey(key, vt)
	case time.Time:
		enc.AddTimeKey(key, &vt, logr.DefTimestampFormat)
	case *time.Time:
		enc.AddTimeKey(key, vt, logr.DefTimestampFormat)
	default:
		s := fmt.Sprintf("%v", vt)
		enc.AddStringKey(key, s)
	}
}
