package wssim

import (
    "fmt"
    "io/ioutil"
    "net/http"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

var Upgrader = websocket.Upgrader{ReadBufferSize: 1024, WriteBufferSize: 1024, CheckOrigin: func(r *http.Request) bool {return true}}
var ActualSimulator *Simulator      // Just allowing one per process.
var NextConnId int

type Message struct {
    Origin                  int
    Text                    string
}

type SimObject interface {  // FOR SAFETY, NONE OF THE FOLLOWING CALLS EVER OCCUR CONCURRENTLY...
    Init()                  //   Called once only, at start
    NewConn(id int)         //   Called when a new WebSocket client arrives, with its unique ID
    Receive(msg Message)    //   Called when a message is received from a client
    CloseConn(id int)       //   Called when a WebSocket is closed, with its unique ID
    Iterate()               //   Called once each frame
    Send(id int) string     //   Called once for each connection each frame; output is the string to send to specified client
}

type Simulator struct {
    simobject               SimObject
    tick_freq               time.Duration
    all_ws_channels         map[int]chan string
    messages                []Message
    new_ids                 []int
    closed_ids              []int
    MUTEX                   sync.Mutex
}

func StartSimulator(server string, path string, simobject SimObject, tick_freq time.Duration) error {

    if ActualSimulator != nil {
        return fmt.Errorf("StartSimulator(): already started a simulation")
    }

    ActualSimulator = new(Simulator)

    ActualSimulator.simobject = simobject
    ActualSimulator.tick_freq = tick_freq
    ActualSimulator.all_ws_channels = make(map[int]chan string)

    go run_sim(ActualSimulator)
    http.HandleFunc(path, ws_handler)

    return nil
}

func ws_handler(writer http.ResponseWriter, request * http.Request) {

    conn, err := Upgrader.Upgrade(writer, request, nil)
    if err != nil {
        return
    }

    our_relay_channel := make(chan string)

    our_id := new_conn_id()

    ActualSimulator.MUTEX.Lock()        // With just one MUTEX for the whole sim, both of these will definitely go through in time...
    ActualSimulator.all_ws_channels[our_id] = our_relay_channel
    ActualSimulator.new_ids = append(ActualSimulator.new_ids, our_id)
    ActualSimulator.MUTEX.Unlock()

    go ws_reader(ActualSimulator, conn, our_id)

    for {
        msg := <- our_relay_channel
        err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
        if err != nil {
            remove_ws_chan(ActualSimulator, our_id)
            return
        }
    }
}

func ws_reader(s *Simulator, conn * websocket.Conn, our_id int) {

    for {
        _, reader, err := conn.NextReader();
        if err != nil {
            remove_ws_chan(s, our_id)
            conn.Close()
            return
        }

        bytes, err := ioutil.ReadAll(reader)
        msg := string(bytes)

        s.MUTEX.Lock()
        s.messages = append(s.messages, Message{our_id, msg})
        s.MUTEX.Unlock()
    }
}

func remove_ws_chan(s *Simulator, id int) {

    s.MUTEX.Lock()
    defer s.MUTEX.Unlock()

    _, ok := s.all_ws_channels[id]
    if ok {
        delete(s.all_ws_channels, id)
        s.closed_ids = append(s.closed_ids, id)
    }
}

func run_sim(s *Simulator) {

    s.simobject.Init()

    ticker := time.Tick(s.tick_freq)

    for {
        <- ticker

        // Disallow all other goroutines from modifying stuff for the duration...

        s.MUTEX.Lock()

        // Tell the simobject about new IDs that connected...

        for _, id := range s.new_ids {
            s.simobject.NewConn(id)
        }
        s.new_ids = s.new_ids[:0]

        // Deal with incoming messages from clients that have been stored...

        for _, incoming := range s.messages {
            s.simobject.Receive(incoming)
        }
        s.messages = s.messages[:0]

        // Tell the simobject about IDs that disconnected...

        for _, id := range s.closed_ids {
            s.simobject.CloseConn(id)
        }
        s.closed_ids = s.closed_ids[:0]

        // Now (maybe) iterate the sim...

        conn_count := len(s.all_ws_channels)

        if conn_count > 0 {     // No simulation without representation!

            // Iterate the simulation...

            s.simobject.Iterate()

            // Send the output to all connected websockets via their goroutine's channel...

            for ws_id, msg_channel := range s.all_ws_channels {
                output := s.simobject.Send(ws_id)
                msg_channel <- output
            }
        }

        // Reallow other goroutines to do stuff...

        s.MUTEX.Unlock()
    }
}

func new_conn_id() int {
    NextConnId += 1
    return NextConnId - 1
}
