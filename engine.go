package wsworld

import (
    "fmt"
    "net/http"
    "path/filepath"
    "strings"
    "sync"
)

const VIRTUAL_RESOURCE_DIR = "/wsworld_resources/"   // Path that the client thinks resources are at.
const VIRTUAL_WS_DIR = "/wsworld_websocket/"         // Path that the client thinks websockets connect to.

var eng engine

func init() {
    eng.sprites = make(map[string]*sprite)
    eng.sounds = make(map[string]*sound)
}

type engine struct {

    mutex           sync.Mutex

    // The following are written once only...

    started         bool
    fps             float64
    res_path_local  string
    title           string
    static          string
    multiplayer     bool

    // The following are written several times at the beginning, then only read from...

    sprites         map[string]*sprite      // filename -> sprite
    sounds          map[string]*sound       // filename -> sound
}

type sprite struct {
    filename        string
    varname         string
}

type sound struct {
    filename        string
    varname         string
}

func RegisterSprite(filename string) {

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    if eng.started {
        panic("RegisterSprite(): already started")
    }

    varname := fmt.Sprintf("sprite%d", len(eng.sprites))

    newsprite := sprite{filename, varname}
    eng.sprites[filename] = &newsprite
}

func RegisterSound(filename string) {

    eng.mutex.Lock()
    defer eng.mutex.Unlock()

    if eng.started {
        panic("RegisterSound(): already started")
    }

    varname := fmt.Sprintf("sound%d", len(eng.sprites))

    newsound := sound{filename, varname}
    eng.sounds[filename] = &newsound
}

func Start(title, server, normal_path, res_path_local string, width, height int, fps float64, multiplayer bool) {

    eng.mutex.Lock()            // Really just for the .started var
    defer eng.mutex.Unlock()

    if eng.started {
        panic("wsengine.Start(): already started")
    }

    eng.started = true

    if res_path_local == "" {
        res_path_local = "not_in_use"
    }

    normal_path = slash_at_both_ends(normal_path)

    eng.title = title
    eng.res_path_local = res_path_local
    eng.fps = fps
    eng.multiplayer = multiplayer

    eng.static = static_webpage(eng.title, server, VIRTUAL_WS_DIR, VIRTUAL_RESOURCE_DIR, eng.sprites, eng.sounds, width, height)

    go http_startup(server, normal_path, VIRTUAL_WS_DIR, VIRTUAL_RESOURCE_DIR, res_path_local)
    go connection_hub()
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
