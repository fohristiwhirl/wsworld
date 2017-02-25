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

func static_webpage(title, server, virtual_ws_path, virtual_res_path string, sprites map[string]*sprite, sounds map[string]*sound, width, height int) string {

    var imageloaders []string
    var soundloaders []string

    for _, sprite := range sprites {
        imageloaders = append(imageloaders, fmt.Sprintf(
            "var %s = new Image(); %s.src = \"http://%s%s%s\";",
            sprite.varname, sprite.varname, server, virtual_res_path, sprite.filename))
    }

    for _, sound := range sounds {
        soundloaders = append(soundloaders, fmt.Sprintf(
            "<audio id=\"%s\" src=\"http://%s%s%s\" preload=\"auto\"></audio>",
            sound.varname, server, virtual_res_path, sound.filename))
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
</p>

{{.SoundLoaders}}

<canvas style="display: block; margin: 0 auto; border-style: dashed; border-color: #666666"></canvas>

<script>
"use strict";

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

var all_things = {}

var ws = new WebSocket("ws://{{.Server}}{{.WsPath}}")

ws.onopen = function (evt) {
    requestAnimationFrame(animate)
}

ws.onmessage = function (evt) {

    var stuff = evt.data.split(" ");
    var frame_type = stuff[0]

    if (frame_type === "v") {

        // Deal with visual frames.................................................................

        ws_frames += 1

        second_last_frame_time = last_frame_time
        last_frame_time = Date.now()

        var len = stuff.length
        for (var n = 1 ; n < len ; n++) {
            parse_point_or_sprite(stuff[n])                 // For now, all objects are sprites or points
        }

        for (var key in all_things) {
            if (all_things[key].last_seen < ws_frames) {    // We didn't see the object, so delete it
                delete all_things[key]
                continue
            }
        }

    } else if (frame_type === "a") {

        // Deal with audio events..................................................................

        var len = stuff.length
        for (var n = 1 ; n < len ; n++) {
            play_multi_sound(stuff[n])
        }
    }
};

function parse_point_or_sprite(s) {

    var elements = s.split(":")
    var id = elements[1]

    var thing

    if (all_things.hasOwnProperty(id) == false) {
        all_things[id] = {}
    }

    thing = all_things[id]

    thing.type = elements[0]
    thing.id = elements[1]

    if (thing.type == "p") {
        thing.colour = elements[2]
    } else if (thing.type == "s") {
        thing.varname = elements[2]
    }

    var frame_x = parseFloat(elements[3])
    var frame_y = parseFloat(elements[4])
    var frame_speedx = parseFloat(elements[5])
    var frame_speedy = parseFloat(elements[6])

    do_latency_compensation(thing, frame_x, frame_y, frame_speedx, frame_speedy)

    thing.last_seen = ws_frames
}

function do_latency_compensation(thing, frame_x, frame_y, frame_speedx, frame_speedy) {

    if (
        all_things.hasOwnProperty("x")      == false ||
        all_things.hasOwnProperty("y")      == false ||
        all_things.hasOwnProperty("speedx") == false ||
        all_things.hasOwnProperty("speedy") == false
    ) {
        thing.x = frame_x
        thing.y = frame_y
        thing.speedx = frame_speedx
        thing.speedy = frame_speedy
        return
    }

    // Here would be our lag compensation, if we had written it.

    var ws_frame_time = last_frame_time - second_last_frame_time

    thing.x = frame_x
    thing.y = frame_y
    thing.speedx = frame_speedx
    thing.speedy = frame_speedy
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

function draw() {

    // As a relatively simple way of dealing with arbitrary timings of incoming data, we
    // always try to draw the object "where it is now" taking into account how long it's
    // been since we received info about it. This is done with this "time_offset" var.

    var time_offset = Date.now() - last_frame_time

    virtue.fillStyle = "black"
    virtue.fillRect(0, 0, {{.Width}}, {{.Height}})

    for (var key in all_things) {

        switch (all_things[key].type) {
        case "p":
            draw_point(all_things[key], time_offset)
            break
        case "s":
            draw_sprite(all_things[key], time_offset)
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
