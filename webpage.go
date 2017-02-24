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
---
Interpolated draws: <span id="interp_draws">0</span>
</p>

{{.SoundLoaders}}

<canvas style="display: block; margin: 0 auto; border-style: dashed; border-color: #666666"></canvas>

<script>
"use strict";

{{.ImageLoaders}}

var WIDTH = {{.Width}}
var HEIGHT = {{.Height}}

var have_drawn_last_ws_frame = false

var ws_frames = 0
var total_draws = 0
var interp_draws = 0

var last_draw_time = Date.now()

var virtue = document.querySelector("canvas").getContext("2d")
document.querySelector("canvas").width = WIDTH
document.querySelector("canvas").height = HEIGHT

var all_things = []

var ws = new WebSocket("ws://{{.Server}}{{.WsPath}}")

ws.onopen = function (evt) {
    requestAnimationFrame(animate)
}

ws.onmessage = function (evt) {

    var stuff = evt.data.split(" ");
    var frame_type = stuff[0]

    if (frame_type === "v") {                               // Visual frames

        ws_frames += 1
        have_drawn_last_ws_frame = false
        all_things.length = 0

        var len = stuff.length
        for (var n = 1 ; n < len ; n++) {

            var new_thing

            switch (stuff[n].charAt(0)) {
            case "p":
                new_thing = parse_point(stuff[n])
                break
            case "s":
                new_thing = parse_sprite(stuff[n])
                break
            }
            all_things.push(new_thing)
        }

    } else if (frame_type === "a") {                        // Audio events

        var len = stuff.length
        for (var n = 1 ; n < len ; n++) {
            play_multi_sound(stuff[n])
        }
    }
};

function parse_point(s) {

    var elements = s.split(":")

    var ret = {}
    ret.type = elements[0]
    ret.id = elements[1]
    ret.colour = elements[2]
    ret.x = parseFloat(elements[3])
    ret.y = parseFloat(elements[4])
    ret.speedx = parseFloat(elements[5])
    ret.speedy = parseFloat(elements[6])

    return ret
}

function draw_point(p) {
    var x = Math.floor(p.x)
    var y = Math.floor(p.y)
    virtue.fillStyle = p.colour
    virtue.fillRect(x, y, 1, 1)
}

function parse_sprite(s) {

    var elements = s.split(":")

    var ret = {}
    ret.type = elements[0]
    ret.id = elements[1]
    ret.varname = elements[2]
    ret.x = parseFloat(elements[3])
    ret.y = parseFloat(elements[4])
    ret.speedx = parseFloat(elements[5])
    ret.speedy = parseFloat(elements[6])

    return ret
}

function draw_sprite(sp) {
    var x = sp.x - window[sp.varname].width / 2
    var y = sp.y - window[sp.varname].height / 2
    virtue.drawImage(window[sp.varname], x, y)
}

function draw() {

    virtue.fillStyle = "black"
    virtue.fillRect(0, 0, {{.Width}}, {{.Height}})

    var len = all_things.length
    for (var n = 0 ; n < len ; n++) {
        switch (all_things[n].type) {
        case "p":
            draw_point(all_things[n])
            break
        case "s":
            draw_sprite(all_things[n])
            break
        }
    }

    last_draw_time = Date.now()
}

function animate() {

    if (have_drawn_last_ws_frame === true) {

        // We didn't receive a websocket message in time, so interpolate...

        if (ws_frames > 0) {
            interp_draws += 1
        }

        var observed_framerate = 1000 / (Date.now() - last_draw_time)

        var len = all_things.length
        for (var n = 0 ; n < len ; n++) {

            all_things[n].x += all_things[n].speedx / observed_framerate
            all_things[n].y += all_things[n].speedy / observed_framerate
        }
    }

    if (ws_frames > 0) {
        total_draws += 1
    }

    if (total_draws % 10 === 0) {
        document.getElementById("ws_frames").innerHTML = ws_frames
        document.getElementById("total_draws").innerHTML = total_draws
        document.getElementById("interp_draws").innerHTML = interp_draws
    }

    draw()
    have_drawn_last_ws_frame = true
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
