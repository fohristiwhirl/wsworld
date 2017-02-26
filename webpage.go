package wsworld

import (
    "bytes"
    "fmt"
    "strings"
    "text/template"         // We can use text version, since all input to the template is trusted
)

type Variables struct {
    Title           string
    Server          string
    WsPath          string
    Width           int
    Height          int
    ImageLoaders    string
    SoundLoaders    string
}

func static_webpage(title, server, virtual_ws_path, virtual_res_path string, sprites map[string]string, sounds map[string]string, width, height int) string {

    var imageloaders []string
    var soundloaders []string

    for filename, varname := range sprites {
        imageloaders = append(imageloaders, fmt.Sprintf(
            "var %s = new Image(); %s.src = \"http://%s%s%s\";",
            varname, varname, server, virtual_res_path, filename))
    }

    for filename, varname := range sounds {
        soundloaders = append(soundloaders, fmt.Sprintf(
            "<audio id=\"%s\" src=\"http://%s%s%s\" preload=\"auto\"></audio>",
            varname, server, virtual_res_path, filename))
    }

    joined_imageloaders := strings.Join(imageloaders, "\n")
    joined_soundloaders := strings.Join(soundloaders, "\n")

    variables := Variables{title, server, virtual_ws_path, width, height, joined_imageloaders, joined_soundloaders}
    t, _ := template.New("static").Parse(WEBPAGE)
    var webpage bytes.Buffer
    t.Execute(&webpage, variables)
    return webpage.String()
}

// Terminology note! A "frame" should always mean a visual websocket message.

const WEBPAGE = `<!DOCTYPE html>
<html>
<head>
<title>{{.Title}}</title>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
</head>
<body style="background-color: black; width: 100%; height: 100%; margin: 0; padding: 0; overflow: hidden;">

<p style="color: #66aa66; text-align: center; margin: 0.2em">
WebSocket frames: <span id="ws_frames">0</span>
---
Total draws: <span id="total_draws">0</span>

<span id="debug_msg"></span>
</p>

{{.SoundLoaders}}

<canvas style="display: block; margin: 0 auto; border-style: dashed; border-color: #666666"></canvas>

<script>
"use strict"

{{.ImageLoaders}}

var WIDTH = {{.Width}}
var HEIGHT = {{.Height}}

var ws_frames = 0
var total_draws = 0

var second_last_frame_time = Date.now() - 16
var last_frame_time = Date.now()

var virtue = document.querySelector("canvas").getContext("2d")
document.querySelector("canvas").width = WIDTH
document.querySelector("canvas").height = HEIGHT

var all_things = []

var ws = new WebSocket("ws://{{.Server}}{{.WsPath}}")

ws.onopen = function (evt) {
    requestAnimationFrame(animate)
}

ws.onmessage = function (evt) {

    var stuff = evt.data.split(" ")
    var frame_type = stuff[0]

    if (frame_type === "v") {

        // Deal with visual frames.................................................................

        all_things.length = 0

        ws_frames += 1

        second_last_frame_time = last_frame_time
        last_frame_time = Date.now()

        var len = stuff.length
        for (var n = 1 ; n < len ; n++) {

            switch (stuff[n].charAt(0)) {

            case "p":
            case "s":
                parse_point_or_sprite(stuff[n])
                break
            case "l":
                parse_line(stuff[n])
                break
            }
        }

    } else if (frame_type === "a") {

        // Deal with audio events..................................................................

        var len = stuff.length
        for (var n = 1 ; n < len ; n++) {
            play_multi_sound(stuff[n])
        }

    } else if (frame_type === "d") {

        // Debug messages..........................................................................

        if (evt.data.length > 2) {
            display_debug_message(evt.data.slice(2))
        }
    }
}

function parse_point_or_sprite(blob) {

    var elements = blob.split(":")

    var thing = {}

    thing.type = elements[0]

    if (thing.type == "p") {
        thing.colour = elements[1]
    } else if (thing.type == "s") {
        thing.varname = elements[1]
    }

    thing.x = parseFloat(elements[2])
    thing.y = parseFloat(elements[3])
    thing.speedx = parseFloat(elements[4])
    thing.speedy = parseFloat(elements[5])

    all_things.push(thing)
}

function parse_line(blob) {

    var elements = blob.split(":")

    var thing = {}

    thing.type = elements[0]
    thing.colour = elements[1]
    thing.x1 = parseFloat(elements[2])
    thing.y1 = parseFloat(elements[3])
    thing.x2 = parseFloat(elements[4])
    thing.y2 = parseFloat(elements[5])
    thing.speedx = parseFloat(elements[6])
    thing.speedy = parseFloat(elements[7])

    all_things.push(thing)
}

function draw_point(p, time_offset) {
    var x = Math.floor(p.x + p.speedx * time_offset / 1000)
    var y = Math.floor(p.y + p.speedy * time_offset / 1000)
    virtue.fillStyle = p.colour
    virtue.fillRect(x, y, 1, 1)
}

function draw_sprite(sp, time_offset) {
    var x = sp.x + sp.speedx * time_offset / 1000
    var y = sp.y + sp.speedy * time_offset / 1000
    virtue.drawImage(window[sp.varname], x - window[sp.varname].width / 2, y - window[sp.varname].height / 2)
}

function draw_line(li, time_offset) {
    var x1 = li.x1 + li.speedx * time_offset / 1000
    var y1 = li.y1 + li.speedy * time_offset / 1000
    var x2 = li.x2 + li.speedx * time_offset / 1000
    var y2 = li.y2 + li.speedy * time_offset / 1000

    virtue.strokeStyle = li.colour
    virtue.beginPath()
    virtue.moveTo(x1, y1)
    virtue.lineTo(x2, y2)
    virtue.stroke()
    virtue.closePath()
}

function draw() {

    // As a relatively simple way of dealing with arbitrary timings of incoming data, we
    // always try to draw the object "where it is now" taking into account how long it's
    // been since we received info about it. This is done with this "time_offset" var.

    var time_offset = Date.now() - last_frame_time

    virtue.clearRect(0, 0, WIDTH, HEIGHT)     // The best way to clear the canvas??

    var len = all_things.length
    for (var n = 0 ; n < len ; n++) {

        switch (all_things[n].type) {
        case "p":
            draw_point(all_things[n], time_offset)
            break
        case "s":
            draw_sprite(all_things[n], time_offset)
            break
        case "l":
            draw_line(all_things[n], time_offset)
            break
        }
    }
}

function animate() {

    if (ws_frames > 0) {
        total_draws += 1
    }

    if (total_draws % 10 === 0) {
        document.getElementById("ws_frames").innerHTML = ws_frames
        document.getElementById("total_draws").innerHTML = total_draws
    }

    draw()
    requestAnimationFrame(animate)
}

function display_debug_message(s) {
    document.getElementById("debug_msg").innerHTML = "--- " + s
}

document.addEventListener("keydown", function(evt) {
    if (evt.key === " ") {
        ws.send("keydown space")
    } else {
        ws.send("keydown " + evt.key)
    }
})

document.addEventListener("keyup", function(evt) {
    if (evt.key === " ") {
        ws.send("keyup space")
    } else {
        ws.send("keyup " + evt.key)
    }
})

// Sound from Thomas Sturm: http://www.storiesinflight.com/html5/audio.html

var channel_max = 10
var audiochannels = new Array()
for (var a = 0 ; a < channel_max ; a++) {
    audiochannels[a] = new Array()
    audiochannels[a]["channel"] = new Audio()
    audiochannels[a]["finished"] = -1
}

function play_multi_sound(s) {
    for (var a = 0 ; a < audiochannels.length ; a++) {
        var thistime = new Date()
        if (audiochannels[a]["finished"] < thistime.getTime()) {
            audiochannels[a]["finished"] = thistime.getTime() + document.getElementById(s).duration * 1000
            audiochannels[a]["channel"].src = document.getElementById(s).src
            audiochannels[a]["channel"].load()
            audiochannels[a]["channel"].play()
            break
        }
    }
}

</script>

</body>
</html>
`
