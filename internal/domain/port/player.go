package port

type PlayerEvent struct {
	Type   string      // e.g. "property-change", "end-file", etc.
	Name   string      // e.g. "time-pos", "volume", etc.
	Data   interface{} // payload
	Error  string
	Reason string
}

type AudioPlayerPort interface {
	Start() error
	Stop()
	LoadFile(url string) error
	SetPause(paused bool) error
	SetVolume(volume int) error
	Seek(seconds float64) error
	Events() <-chan PlayerEvent
}
