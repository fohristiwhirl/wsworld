package wsworld

// PRINCIPLES:
// Adding entities to a canvas should be thread-safe.
// Operating on different entities concurrently should be thread-safe.
// Operating on the same entity concurrently is NOT thread-safe.
// Operating on an entity that has been removed from the map is harmless.
//
// THEREFORE:
// Accessing the canvas requires a mutex.
// But no mutex is needed for entities.

import (
    "fmt"
    "strings"
    "sync"

    "github.com/gorilla/websocket"
)

var next_entity_id int

type Entity struct {
    X           float64
    Y           float64
    Speedx      float64
    Speedy      float64
    Colour      string      // For points etc
    Hidden      bool

    id          int
    c           rune        // What sort of thing this is
    filename    string      // For sprites only
    canvas      *Canvas
}

func (e *Entity) Move() {
    e.X += e.Speedx
    e.Y += e.Speedy
}

func (e *Entity) Remove() {
    e.canvas.mutex.Lock()
    defer e.canvas.mutex.Unlock()
    delete(e.canvas.entities, e.id)
}

func (e *Entity) Exists() bool {
    e.canvas.mutex.Lock()
    defer e.canvas.mutex.Unlock()
    _, ok := e.canvas.entities[e.id]
    return ok
}

type Canvas struct {
    mutex sync.Mutex
    entities map[int]*Entity
    soundqueue []string
}

func NewCanvas() *Canvas {
    ret := new(Canvas)
    ret.entities = make(map[int]*Entity)
    return ret
}

func (w *Canvas) new_entity(x, y, speedx, speedy float64, colour string, c rune, filename string) *Entity {

    w.mutex.Lock()
    defer w.mutex.Unlock()

    new_ent := Entity{
        X: x,
        Y: y,
        Speedx: speedx,
        Speedy: speedy,
        Colour: colour,
        Hidden: false,

        id: next_entity_id,
        c: c,
        filename: filename,
        canvas: w,
    }

    w.entities[next_entity_id] = &new_ent

    next_entity_id++

    return &new_ent
}

func (w *Canvas) NewPoint(colour string, x, y, speedx, speedy float64) *Entity {
    return w.new_entity(x, y, speedx, speedy, colour, 'p', "")
}

func (w *Canvas) NewSprite(filename string, x, y, speedx, speedy float64) *Entity {
    return w.new_entity(x, y, speedx, speedy, "", 's', filename)
}

func (w *Canvas) PlaySound(filename string) {

    w.mutex.Lock()
    defer w.mutex.Unlock()

    if len(w.soundqueue) >= 32 {
        return
    }

    sound := eng.sounds[filename]       // Safe to read without mutex since there are no writes any more
    if sound == nil {
        return
    }
    varname := sound.varname
    w.soundqueue = append(w.soundqueue, varname)
}

func (w *Canvas) Send() error {

    fps := eng.fps                                  // Safe to read without mutex since there are no writes any more

    w.mutex.Lock()

    var visual_slice = []string{"v"}                // Header: "v" for "visual"

    for _, e := range w.entities {

        if e.Hidden {
            continue
        }

        switch e.c {
        case 's':
            sprite := eng.sprites[e.filename]       // Safe to read without mutex since there are no writes any more

            var varname string
            if sprite != nil {
                varname = sprite.varname
            }

            visual_slice = append(visual_slice,
                        fmt.Sprintf("s:i%d:%s:%.1f:%.1f:%.1f:%.1f", e.id, varname, e.X, e.Y, e.Speedx * fps, e.Speedy * fps))
        case 'p':
            visual_slice = append(visual_slice,
                        fmt.Sprintf("p:i%d:%s:%.1f:%.1f:%.1f:%.1f", e.id, e.Colour, e.X, e.Y, e.Speedx * fps, e.Speedy * fps))
        }
    }

    w.mutex.Unlock()

    visual_message := strings.Join(visual_slice, " ")

    // Now do sounds, which are easy to assemble...

    sound_message := "a " + strings.Join(w.soundqueue, " ")      // Header: "a" for "audio"

    // Send both...

    var err error

    eng.mutex.Lock()
    if (eng.conn == nil) {

        err = fmt.Errorf("connection was nil")

    } else {

        if len(w.soundqueue) > 0 {
            err = eng.conn.WriteMessage(websocket.TextMessage, []byte(sound_message))
        }

        // If the audio send succeeded or wasn't attempted, we can also sent video...

        if err == nil {
            err = eng.conn.WriteMessage(websocket.TextMessage, []byte(visual_message))
        }
    }
    eng.mutex.Unlock()

    w.mutex.Lock()
    w.soundqueue = nil      // Always clear the sound queue regardless of send...
    w.mutex.Unlock()

    if err != nil {
        return fmt.Errorf("Send(): %v", err)
    }

    return nil
}
