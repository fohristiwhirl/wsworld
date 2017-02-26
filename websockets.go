package wsworld

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "strings"
    "sync"

    "github.com/gorilla/websocket"
)

type safe_counter_struct struct {
    i       int
    mutex   sync.Mutex
}

func (sc *safe_counter_struct) Next() int {
    sc.mutex.Lock()
    defer sc.mutex.Unlock()
    sc.i += 1
    return sc.i - 1
}

var player_id_counter safe_counter_struct


func ws_handler(writer http.ResponseWriter, request * http.Request) {

    fmt.Printf("Connection opened: %s\n", request.RemoteAddr)

    var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(r *http.Request) bool {return true}}

    conn, err := upgrader.Upgrade(writer, request, nil)
    if err != nil {
        fmt.Printf("Upgrade failed: %v\n", err)
        return
    }

    pid := player_id_counter.Next()

    eng.mutex.Lock()

    if eng.multiplayer == false {
        delete(eng.players, eng.latest_player)
    }

    keyboard := make(map[string]bool)
    eng.players[pid] = &player{pid, keyboard, conn}
    eng.latest_player = pid

    eng.mutex.Unlock()

    // Handle incoming messages until connection fails...

    for {
        _, reader, err := conn.NextReader()

        if err != nil {

            conn.Close()
            fmt.Printf("Connection CLOSED: %s (%v)\n", request.RemoteAddr, err)

            eng.mutex.Lock()
            delete(eng.players, pid)
            eng.mutex.Unlock()

            return
        }

        bytes, err := ioutil.ReadAll(reader)        // FIXME: this may be vulnerable to malicious huge messages

        fields := strings.Fields(string(bytes))

        switch fields[0] {

        case "keyup":

            if len(fields) > 1 {
                eng.mutex.Lock()
                if eng.players[pid] != nil {
                    eng.players[pid].keyboard[fields[1]] = false
                }
                eng.mutex.Unlock()
            }

        case "keydown":

            if len(fields) > 1 {
                eng.mutex.Lock()
                if eng.players[pid] != nil {
                    eng.players[pid].keyboard[fields[1]] = true
                }
                eng.mutex.Unlock()
            }
        }
    }
}
