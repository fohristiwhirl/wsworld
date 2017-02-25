package wsengine

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "path/filepath"
    "strings"
    "sync"

    "github.com/gorilla/websocket"
)

var eng engine  // A singleton; we only support 1 engine.
var upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(r *http.Request) bool {return true}}
var next_conn_id int

type engine struct {
    started         bool
    static          string
    players         map[int]*player
    sprites         []*sprite
    res_path_local  string
    mutex           sync.Mutex
}

type player struct {
    id              int
    framecount      int                 // How many messages this player has received
    conn            *websocket.Conn
    keyboard        map[string]bool
}

type sprite struct {
    filename        string
    varname         string
    width           int
    height          int
}

func RegisterSprite(filename string, width, height int) error {

    if eng.started {
        return fmt.Errorf("wsengine.RegisterSprite(): already started")
    }

    varname := varname_from_filename(filename)

    newsprite := sprite{filename, varname, width, height}
    eng.sprites = append(eng.sprites, &newsprite)

    return nil
}

func Start(server, normal_path, ws_path, res_path_server, res_path_local string, width, height int) error {

    if eng.started {
        return fmt.Errorf("wsengine.Start(): already started")
    }

    if res_path_server == "" {
        res_path_server = "not_in_use"
    }
    if res_path_local == "" {
        res_path_local = "not_in_use"
    }

    normal_path = slash_at_both_ends(normal_path)
    ws_path = slash_at_both_ends(ws_path)
    res_path_server = slash_at_both_ends(res_path_server)

    eng.started = true
    eng.players = make(map[int]*player)
    eng.res_path_local = res_path_local

    eng.static = static_webpage(server, ws_path, res_path_server, eng.sprites, width, height)

    go http_startup(server, normal_path, ws_path, res_path_server, res_path_local)

    return nil
}

func ConnectionSet() map[int]bool {        // Returns a simple set of connected players; pid -> true

    var ret map[int]bool = make(map[int]bool)

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    for key, _ := range eng.players {
        ret[key] = true
    }

    return ret
}

func TotalConnections() int {
    eng.mutex.Lock()
    defer eng.mutex.Unlock()
    return len(eng.players)
}

func IsConnected(id int) bool {

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    _, ok := eng.players[id]
    return ok
}

func KeyDown(id int, key string) bool {

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    p, ok := eng.players[id]
    if ok == false {
        return false
    }

    return p.keyboard[key]
}

func http_startup(server, normal_path, ws_path, res_path_server, res_path_local string) {

    // FIXME: how safe is the following, exactly?

    var pass_to_servefile = func(writer http.ResponseWriter, request * http.Request) {

        // Note that this only works for resource files at the base level of the dir

        p := filepath.Base(request.URL.Path)
        http.ServeFile(writer, request, filepath.Join(res_path_local, p))
    }

    http.HandleFunc(ws_path, ws_handler)
    http.HandleFunc(res_path_server, pass_to_servefile)
    http.HandleFunc(normal_path, normal_handler)
    http.ListenAndServe(server, nil)
}

func ws_handler(writer http.ResponseWriter, request * http.Request) {

    conn, err := upgrader.Upgrade(writer, request, nil)
    if err != nil {
        return
    }

    our_id := new_conn_id()

    p := new(player)
    p.id = our_id
    p.conn = conn
    p.keyboard = make(map[string]bool)

    eng.mutex.Lock()
    eng.players[our_id] = p
    eng.mutex.Unlock()

    for {
        _, reader, err := conn.NextReader();
        if err != nil {
            delete_player(our_id)
            return
        }

        bytes, err := ioutil.ReadAll(reader)        // FIXME: this may be vulnerable to malicious huge messages

        handle_message(our_id, string(bytes))
    }
}

func handle_message(id int, msg string) {

    fields := strings.Fields(msg)

    if len(fields) < 2 {
        return
    }

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    switch fields[0] {
    case "keyup":
        p, ok := eng.players[id]
        if ok {
            p.keyboard[fields[1]] = false
        }
    case "keydown":
        p, ok := eng.players[id]
        if ok {
            p.keyboard[fields[1]] = true
        }
    }
}

func delete_player(id int) error {

    // Never call this if the mutex is already locked!

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    p, ok := eng.players[id]
    if ok {
        p.conn.Close()
        delete(eng.players, id)
        return nil
    }

    return fmt.Errorf("delete_player(%d): not in players map", id)
}

func new_conn_id() int {
    next_conn_id += 1
    return next_conn_id - 1
}

func normal_handler(writer http.ResponseWriter, request * http.Request) {
    writer.Write([]byte(eng.static))   // Created in file webpage.go
}

func slash_at_both_ends(s string) string {
    if strings.HasPrefix(s, "/") == false {
        s = "/" + s
    }
    if strings.HasSuffix(s, "/") == false {
        s = s + "/"
    }
    return s
}

func varname_from_filename(filename string) string {
    return strings.Replace(filename, ".", "", -1)           // What the Javascript will call it (strip periods)
}
