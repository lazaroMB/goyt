package mpv

import (
	"goyt/internal/domain/port"
	"goyt/pkg/player"
)

type MpvPlayerAdapter struct {
	p          *player.Player
	eventsChan chan port.PlayerEvent
}

func NewMpvPlayerAdapter() *MpvPlayerAdapter {
	p := player.NewPlayer()
	a := &MpvPlayerAdapter{
		p:          p,
		eventsChan: make(chan port.PlayerEvent, 100),
	}
	return a
}

func (a *MpvPlayerAdapter) Start() error {
	if err := a.p.Start(); err != nil {
		return err
	}
	go a.forwardEvents()
	return nil
}

func (a *MpvPlayerAdapter) Stop() {
	a.p.Stop()
}

func (a *MpvPlayerAdapter) LoadFile(url string) error {
	return a.p.LoadFile(url)
}

func (a *MpvPlayerAdapter) SetPause(paused bool) error {
	return a.p.SetPause(paused)
}

func (a *MpvPlayerAdapter) SetVolume(volume int) error {
	return a.p.SetVolume(volume)
}

func (a *MpvPlayerAdapter) Seek(seconds float64) error {
	return a.p.Seek(seconds)
}

func (a *MpvPlayerAdapter) Events() <-chan port.PlayerEvent {
	return a.eventsChan
}

func (a *MpvPlayerAdapter) forwardEvents() {
	for ev := range a.p.Events() {
		a.eventsChan <- port.PlayerEvent{
			Type:   ev.Event,
			Name:   ev.Name,
			Data:   ev.Data,
			Error:  ev.Error,
			Reason: ev.Reason,
		}
	}
}
