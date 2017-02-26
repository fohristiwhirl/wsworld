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
    return ret
}

func (w *Canvas) Clear() {
    w.mutex.Lock()
    defer w.mutex.Unlock()
    w.entities = nil
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

    visual_message := []byte("v " + strings.Join(w.entities, " "))      // Header: "v" for "visual"
    sound_message := []byte("a " + strings.Join(w.soundqueue, " "))     // Header: "a" for "audio"

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
        w.mutex.Lock()
        w.soundqueue = nil
        w.mutex.Unlock()
    }
}
