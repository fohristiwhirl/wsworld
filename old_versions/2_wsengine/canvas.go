package wsengine

import (
    "fmt"
    "strings"

    "github.com/gorilla/websocket"
)

type Canvas struct {                // The only exported type!
    strings     []string
}

func (c *Canvas) Send(id int) error {

    main_message := strings.Join(c.strings, " ")

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    p, ok := eng.players[id]
    if ok == false {
        return fmt.Errorf("Send(%d): not in players map", id)
    }

    p.framecount += 1
    framecount_string := fmt.Sprintf("%d", p.framecount)

    final_message_slice := []string{framecount_string, main_message}
    message := strings.Join(final_message_slice, " ")

    err := p.conn.WriteMessage(websocket.TextMessage, []byte(message))
    if err != nil {
        eng.mutex.Unlock()      // Note that delete_player() will lock and unlock also.
        delete_player(p.id)     // Will lock and unlock.
        eng.mutex.Lock()        // Because an unlock is deferred.
        return fmt.Errorf("Send(%d): %v", id, err)
    }

    return nil
}

func (c *Canvas) SendToAll() {

    main_message := strings.Join(c.strings, " ")

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    for _, p := range eng.players {

        p.framecount += 1
        framecount_string := fmt.Sprintf("%d", p.framecount)

        final_message_slice := []string{framecount_string, main_message}
        message := strings.Join(final_message_slice, " ")

        err := p.conn.WriteMessage(websocket.TextMessage, []byte(message))
        if err != nil {
            eng.mutex.Unlock()      // Yuk, gotta unlock so delete_player() can lock. FIXME?
            delete_player(p.id)     // Apparently it's safe to delete from a map while ranging over it
            eng.mutex.Lock()
        }
    }
}

func (c *Canvas) Clear() {
    c.strings = c.strings[:0]
}

func (c *Canvas) AddPoint(colour string, x, y, speedx, speedy float64, fps int) {

    // The client receives a speed that is distance per second...

    speedx *= float64(fps)
    speedy *= float64(fps)

    c.strings = append(c.strings, fmt.Sprintf("p:%s:%.2f:%.2f:%.2f:%.2f", colour, x, y, speedx, speedy))
}

func (c *Canvas) AddCircle(colour string, radius int, x, y, speedx, speedy float64, fps int) {

    // The client receives a speed that is distance per second...

    speedx *= float64(fps)
    speedy *= float64(fps)

    c.strings = append(c.strings, fmt.Sprintf("c:%s:%d:%.2f:%.2f:%.2f:%.2f", colour, radius, x, y, speedx, speedy))
}

func (c *Canvas) AddSprite(filename string, x, y, speedx, speedy float64, fps int) {

    speedx *= float64(fps)
    speedy *= float64(fps)

    varname := varname_from_filename(filename)

    c.strings = append(c.strings, fmt.Sprintf("s:%s:%.2f:%.2f:%.2f:%.2f", varname, x, y, speedx, speedy))
}
