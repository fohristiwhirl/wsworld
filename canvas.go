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
    soundqueue      []string
}

func NewCanvas() *Canvas {
    ret := new(Canvas)
    ret.Clear()
    ret.ClearSounds()
    return ret
}

func (w *Canvas) Clear() {
    w.mutex.Lock()
    defer w.mutex.Unlock()
    w.entities = []string{"v"}
}

func (w *Canvas) ClearSounds() {
    w.mutex.Lock()
    defer w.mutex.Unlock()
    w.soundqueue = []string{"a"}
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

func (w *Canvas) PlaySound(filename string) {

    w.mutex.Lock()
    defer w.mutex.Unlock()

    if len(w.soundqueue) >= 32 {
        return
    }

    sound := eng.sounds[filename]                   // Safe to read without mutex since there are no writes any more
    if sound == nil {
        return
    }
    varname := sound.varname
    w.soundqueue = append(w.soundqueue, varname)
}

func (w *Canvas) SendToAll() {

    w.mutex.Lock()

    var havesounds bool

    if len(w.soundqueue) > 0{
        havesounds = true
    }

    visual_message := []byte(strings.Join(w.entities, " "))
    sound_message := []byte(strings.Join(w.soundqueue, " "))

    w.mutex.Unlock()

    // Send both...

    eng.mutex.Lock()

    for _, player := range eng.players {

        if havesounds {
            player.conn.WriteMessage(websocket.TextMessage, sound_message)
        }

        player.conn.WriteMessage(websocket.TextMessage, visual_message)
    }

    eng.mutex.Unlock()

    if havesounds {
        w.ClearSounds()
    }
}
