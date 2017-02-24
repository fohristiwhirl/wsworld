package wsworld

// PRINCIPLES:
// Adding entities to a canvas should be thread-safe.
// Operating on different entities concurrently should be thread-safe.
// Operating on the same entity concurrently is NOT thread-safe.
// Operating on an entity that has been removed from the map is harmless.
//
// THEREFORE:
// Accessing the canvas map requires a mutex.
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
    fps int
    mutex sync.Mutex
    entities map[int]*Entity
}

func NewCanvas(fps int) *Canvas {
    ret := new(Canvas)
    ret.fps = fps
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

func (w *Canvas) Send() error {

    var main_message_slice []string

    w.mutex.Lock()

    for _, e := range w.entities {

        if e.Hidden {
            continue
        }

        switch e.c {
        case 's':
            sprite := eng.sprites[e.filename]     // Safe to read without mutex since there are no writes any more

            var varname string
            if sprite != nil {
                varname = sprite.varname
            }

            main_message_slice = append(main_message_slice,
                        fmt.Sprintf("s:%s:%.1f:%.1f:%.1f:%.1f", varname, e.X, e.Y, e.Speedx * float64(w.fps), e.Speedy * float64(w.fps)))
        case 'p':
            main_message_slice = append(main_message_slice,
                        fmt.Sprintf("p:%s:%.1f:%.1f:%.1f:%.1f", e.Colour, e.X, e.Y, e.Speedx * float64(w.fps), e.Speedy * float64(w.fps)))
        }
    }

    w.mutex.Unlock()

    main_message := strings.Join(main_message_slice, " ")

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    if eng.conn == nil {
        return fmt.Errorf("Send(): no connection")
    }

    eng.framecount += 1
    header_string := fmt.Sprintf("s %d", eng.framecount)        // Header is "s" for sprites and then a counter
    actual_message_slice := []string{header_string, main_message}
    message := strings.Join(actual_message_slice, " ")

    err := eng.conn.WriteMessage(websocket.TextMessage, []byte(message))
    if err != nil {
        return fmt.Errorf("Send(): %v", err)
    }

    return nil
}
