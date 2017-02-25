package wsworld

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "strings"
    "sync"

    "github.com/gorilla/websocket"
)

var player_count int
var player_count_mutex sync.Mutex

func new_player_id() int {
    player_count_mutex.Lock()
    defer player_count_mutex.Unlock()
    player_count++
    return player_count - 1
}

func ws_handler(writer http.ResponseWriter, request * http.Request) {

    fmt.Printf("Connection opened: %s\n", request.RemoteAddr)

    var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(r *http.Request) bool {return true}}

    conn, err := upgrader.Upgrade(writer, request, nil)
    if err != nil {
        return
    }

    my_outgoing_msg_chan := make(chan string, 16)
    pid := new_player_id()

    new_player_chan <- new_player{pid, my_outgoing_msg_chan}

    go incoming_msg_handler(pid, conn, request.RemoteAddr)

    for {
        m := <- my_outgoing_msg_chan
        err := conn.WriteMessage(websocket.TextMessage, []byte(m))

        if err != nil {
            conn.Close()
            remove_player_chan <- pid
            return
        }
    }
}

func incoming_msg_handler(pid int, conn *websocket.Conn, remote_address string) {

    for {
        _, reader, err := conn.NextReader()

        if err != nil {
            conn.Close()
            fmt.Printf("Connection CLOSED: %s (%v)\n", remote_address, err)
            remove_player_chan <- pid
            return
        }

        bytes, err := ioutil.ReadAll(reader)        // FIXME: this may be vulnerable to malicious huge messages

        fields := strings.Fields(string(bytes))

        switch fields[0] {

        case "keyup":

            if len(fields) > 1 {
                key_input_chan <- key_input{pid, fields[1], false}
            }

        case "keydown":

            if len(fields) > 1 {
                key_input_chan <- key_input{pid, fields[1], true}
            }
        }
    }
}
