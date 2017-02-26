package wsworld

import (
    "fmt"
    "strings"
    "sync"

    "github.com/gorilla/websocket"
)


type Canvas struct {
    mutex           sync.Mutex
    entities        []string
}
func NewCanvas() *Canvas {
    ret := new(Canvas)
    ret.Clear()
    return ret
}
func (w *Canvas) Clear() {
    w.mutex.Lock()
    defer w.mutex.Unlock()
    w.entities = []string{"v"}
}
func (w *Canvas) Bytes() []byte {
    w.mutex.Lock()
    defer w.mutex.Unlock()
    return []byte(strings.Join(w.entities, " "))
}


type Soundscape struct {
    mutex           sync.Mutex
    soundqueue      []string
}
func NewSoundscape() *Soundscape {
    ret := new(Soundscape)
    ret.Clear()
    return ret
}
func (z *Soundscape) Clear() {
    z.mutex.Lock()
    defer z.mutex.Unlock()
    z.soundqueue = []string{"a"}
}
func (z *Soundscape) Bytes() []byte {
    z.mutex.Lock()
    defer z.mutex.Unlock()
    return []byte(strings.Join(z.soundqueue, " "))
}


func (w *Canvas) AddPoint(colour string, x, y, speedx, speedy float64) {
    w.mutex.Lock()
    defer w.mutex.Unlock()
    w.entities = append(w.entities, fmt.Sprintf("p:%s:%.1f:%.1f:%.1f:%.1f", colour, x, y, speedx * eng.fps, speedy * eng.fps))
}

func (w *Canvas) AddSprite(filename string, x, y, speedx, speedy float64) {
    w.mutex.Lock()
    defer w.mutex.Unlock()
    varname := eng.sprites[filename].varname        // Safe to read without mutex since there are no writes any more
    w.entities = append(w.entities, fmt.Sprintf("s:%s:%.1f:%.1f:%.1f:%.1f", varname, x, y, speedx * eng.fps, speedy * eng.fps))
}

func (z *Soundscape) PlaySound(filename string) {

    z.mutex.Lock()
    defer z.mutex.Unlock()

    if len(z.soundqueue) >= 32 {
        return
    }

    sound := eng.sounds[filename]                   // Safe to read without mutex since there are no writes any more
    if sound == nil {
        return
    }
    varname := sound.varname
    z.soundqueue = append(z.soundqueue, varname)
}

func (w *Canvas) SendToAll() {

    visual_message := w.Bytes()

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    for _, player := range eng.players {
        player.conn.WriteMessage(websocket.TextMessage, visual_message)
    }
}

func (z *Soundscape) SendToAll() {

    z.mutex.Lock()
    queue_length := len(z.soundqueue)
    z.mutex.Unlock()

    if queue_length < 2 {
        return
    }

    sound_message := z.Bytes()  // Although the queue length may have changed (race condition), that's harmless enough.

    eng.mutex.Lock()
    for _, player := range eng.players {
        player.conn.WriteMessage(websocket.TextMessage, sound_message)
    }
    eng.mutex.Unlock()
}
