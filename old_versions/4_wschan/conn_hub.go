package wsworld

type new_player struct {
    pid             int
    message_chan    chan string
}

type out_message struct {
    pid             int
    message         string
}

type key_input struct {
    pid             int
    key             string
    is_down         bool
}

type key_query struct {
    pid             int
    key             string
    response_chan   chan bool
}

// Internal comm channels...

var remove_player_chan chan int = make(chan int, 16)
var new_player_chan chan new_player = make(chan new_player, 16)
var message_relay_chan chan out_message = make(chan out_message, 16)
var universal_message_chan chan string = make(chan string, 16)
var key_input_chan chan key_input = make(chan key_input, 16)

// Queries from app...

var key_query_chan chan key_query = make(chan key_query)
var conn_count_query_chan chan chan int = make(chan chan int)
var conn_set_query_chan chan chan map[int]bool = make(chan chan map[int]bool)

func connection_hub() {

    type player struct {
        pid             int
        message_chan    chan string
        keyboard        map[string]bool
    }

    known_players := make(map[int]*player)

    var latest_player int

    for {

        select {

        // A connection was closed........................................
        case pid := <- remove_player_chan:

            delete(known_players, pid)

        // A connection was opened........................................
        case np := <- new_player_chan:

            if eng.multiplayer == false {
                delete(known_players, latest_player)
            }

            keyboard := make(map[string]bool)
            known_players[np.pid] = &player{np.pid, np.message_chan, keyboard}
            latest_player = np.pid

        // A message was received for a client............................
        case ms := <- message_relay_chan:

            p, ok := known_players[ms.pid]

            if ok {
                select {                                // Don't send if we somehow can't. Should be impossible with our big buffer.
                case p.message_chan <- ms.message:
                default:
                }
            }

        // A message was received for all clients.........................
        case universal_msg := <- universal_message_chan:

            for _, p := range known_players {
                select {
                case p.message_chan <- universal_msg:
                default:
                }
            }

        // Key input was received.........................................
        case k := <- key_input_chan:

            p, ok := known_players[k.pid]

            if ok {
                p.keyboard[k.key] = k.is_down
            }

        // Key was queried................................................
        case key_query := <- key_query_chan:

            if key_query.pid == -1 {
                key_query.pid = latest_player
            }

            p, ok := known_players[key_query.pid]

            if ok {
                key_query.response_chan <- p.keyboard[key_query.key]
            } else {
                key_query.response_chan <- false
            }

        // Request for player count.......................................
        case reply_chan := <- conn_count_query_chan:

            reply_chan <- len(known_players)

        // Request for player map.........................................
        case reply_chan := <- conn_set_query_chan:

            set := make(map[int]bool)

            for key, _ := range known_players {

                set[key] = true
            }

            reply_chan <- set
        }
    }
}

func KeyDown(pid int, key string) bool {
    response_chan := make(chan bool, 1)
    q := key_query{pid, key, response_chan}
    key_query_chan <- q
    return <- response_chan
}

func PlayerCount() int {
    response_chan := make(chan int, 1)
    conn_count_query_chan <- response_chan
    return <- response_chan
}

func PlayerSet() map[int]bool {
    response_chan := make(chan map[int]bool, 1)
    conn_set_query_chan <- response_chan
    return <- response_chan
}
