package wsloco

import (
    "fmt"
    "strings"

    "github.com/gorilla/websocket"
)

type Canvas struct {                // The only exported type!
    strings         []string
    sound_strings   []string
}

func (c *Canvas) Send() error {

    main_message := strings.Join(c.strings, " ")
    sound_message := strings.Join(c.sound_strings, " ")

    // --------------- Sprites

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

    // --------------- Sounds

    if len(c.sound_strings) == 0 {
        return nil
    }

    header_string = fmt.Sprintf("a %d", eng.framecount)        // Header is "a" for audio and then the same counter (but it's ignored)

    actual_message_slice = []string{header_string, sound_message}
    message = strings.Join(actual_message_slice, " ")

    err = eng.conn.WriteMessage(websocket.TextMessage, []byte(message))
    if err != nil {
        return fmt.Errorf("Send(): %v", err)
    }

    return nil
}

func (c *Canvas) Clear() {
    c.strings = c.strings[:0]
    c.sound_strings = c.sound_strings[:0]
}

func (c *Canvas) AddPoint(colour string, x, y, speedx, speedy float64) {

    speedx *= fps
    speedy *= fps

    c.strings = append(c.strings, fmt.Sprintf("p:%s:%.1f:%.1f:%.1f:%.1f", colour, x, y, speedx, speedy))
}

func (c *Canvas) AddCircle(colour string, radius int, x, y, speedx, speedy float64) {

    speedx *= fps
    speedy *= fps

    c.strings = append(c.strings, fmt.Sprintf("c:%s:%d:%.1f:%.1f:%.1f:%.1f", colour, radius, x, y, speedx, speedy))
}

func (c *Canvas) AddSprite(filename string, x, y, speedx, speedy float64) {

    speedx *= fps
    speedy *= fps

    sprite := eng.sprites[filename]     // Safe to read without mutex since there are no writes any more
    if sprite == nil {
        return
    }
    varname := sprite.varname

    c.strings = append(c.strings, fmt.Sprintf("s:%s:%.1f:%.1f:%.1f:%.1f", varname, x, y, speedx, speedy))
}

func (c *Canvas) AddSound(filename string) {

    sound := eng.sounds[filename]       // Safe to read without mutex since there are no writes any more
    if sound == nil {
        return
    }
    varname := sound.varname

    c.sound_strings = append(c.sound_strings, fmt.Sprintf("a:%s", varname))
}
