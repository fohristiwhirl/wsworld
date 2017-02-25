package wsengine

import (
    "bytes"
    "fmt"
    "strings"
    "text/template"         // We can use text version, since all input to the template is trusted
)

type Variables struct {
    Server          string
    WsPath          string
    Width           int
    Height          int
    ImageLoaders    string
}

func static_webpage(server, ws_path, res_path_server string, sprites []*sprite, width, height int) string {

    var imageloaders []string

    for _, sprite := range sprites {
        imageloaders = append(imageloaders, fmt.Sprintf(
            "%s = new Image(); %s.src = \"http://%s%s%s\"; %s.claimedwidth = %d; %s.claimedheight = %d",
            sprite.varname, sprite.varname, server, res_path_server, sprite.filename, sprite.varname, sprite.width, sprite.varname, sprite.height))
    }

    joined_imageloaders := strings.Join(imageloaders, "\n")

    variables := Variables{server, ws_path, width, height, joined_imageloaders}
    t, _ := template.New("static").Parse(WEBPAGE)
    var webpage bytes.Buffer
    t.Execute(&webpage, variables)
    return webpage.String()
}

const WEBPAGE = `<!DOCTYPE html>
<html>
<head>
<title>WS Engine</title>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8">
</head>
<body style="background-color: black; width: 100%; height: 100%; margin: 0; padding: 0; overflow: hidden;">

<br />

<canvas style="display: block; margin: 0 auto; border-style: dashed; border-color: #666666"></canvas>

<script>

{{.ImageLoaders}}

WIDTH = {{.Width}}
HEIGHT = {{.Height}}

last_frame_received = -1
last_frame_drawn = -1

last_draw_time = Date.now()

virtue = document.querySelector("canvas").getContext("2d");
document.querySelector("canvas").width = WIDTH
document.querySelector("canvas").height = HEIGHT

all_things = []

ws = new WebSocket("ws://{{.Server}}{{.WsPath}}");

ws.onmessage = function (evt)
{
    all_things = []
    var stuff = evt.data.split(" ");

    last_frame_received = stuff[0]

    var len = stuff.length
    for (var n = 1 ; n < len ; n++) {

        var new_thing

        switch (stuff[n].charAt(0)) {
        case "p":
            new_thing = parse_point(stuff[n])
            break
        case "c":
            new_thing = parse_circle(stuff[n])
            break
        case "s":
            new_thing = parse_sprite(stuff[n])
            break
        }
        all_things.push(new_thing)
    }
};

function parse_point(s) {

    var elements = s.split(":")

    var ret = {}
    ret.type = elements[0]
    ret.colour = elements[1]
    ret.x = parseFloat(elements[2])
    ret.y = parseFloat(elements[3])
    ret.speedx = parseFloat(elements[4])
    ret.speedy = parseFloat(elements[5])

    return ret
}

function draw_point(p) {
    var x = Math.floor(p.x)
    var y = Math.floor(p.y)
    virtue.fillStyle = p.colour
    virtue.fillRect(x, y, 1, 1)
}

function parse_circle(s) {

    var elements = s.split(":")

    var ret = {}
    ret.type = elements[0]
    ret.colour = elements[1]
    ret.radius = elements[2]
    ret.x = parseFloat(elements[3])
    ret.y = parseFloat(elements[4])
    ret.speedx = parseFloat(elements[5])
    ret.speedy = parseFloat(elements[6])

    return ret
}

function draw_circle(c) {
    virtue.beginPath()
    virtue.arc(c.x, c.y, c.radius, 0, 2 * Math.PI, false)
    virtue.fillStyle = c.colour
    virtue.fill()
}

function parse_sprite(s) {

    var elements = s.split(":")

    var ret = {}
    ret.type = elements[0]
    ret.varname = elements[1]
    ret.x = parseFloat(elements[2])
    ret.y = parseFloat(elements[3])
    ret.speedx = parseFloat(elements[4])
    ret.speedy = parseFloat(elements[5])

    return ret
}

function draw_sprite(sp) {
    var x = sp.x - window[sp.varname].claimedwidth / 2
    var y = sp.y - window[sp.varname].claimedheight / 2
    virtue.drawImage(window[sp.varname], x, y)
}

function draw() {
    virtue.fillStyle = "black"
    virtue.fillRect(0, 0, {{.Width}}, {{.Height}})

    for (var n = 0 ; n < all_things.length ; n++) {
        switch (all_things[n].type) {
        case "p":
            draw_point(all_things[n])
            break
        case "c":
            draw_circle(all_things[n])
            break
        case "s":
            draw_sprite(all_things[n])
            break
        }
    }

    last_frame_drawn = last_frame_received
    last_draw_time = Date.now()
}

function animate() {

    if (last_frame_received <= last_frame_drawn) {

        // We didn't receive a websocket message in time, so interpolate...

        var observed_framerate = 1000 / (Date.now() - last_draw_time)

        var len = all_things.length
        for (var n = 0 ; n < len ; n++) {

            all_things[n].x += all_things[n].speedx / observed_framerate
            all_things[n].y += all_things[n].speedy / observed_framerate
        }
    }

    draw()

    requestAnimationFrame(animate)
}

document.addEventListener('keydown', function(evt) {
    if (evt.key == " ") {
        ws.send("keydown space")
    } else {
        ws.send("keydown " + evt.key)
    }
})

document.addEventListener('keyup', function(evt) {
    if (evt.key == " ") {
        ws.send("keyup space")
    } else {
        ws.send("keyup " + evt.key)
    }
})

requestAnimationFrame(animate)

</script>

</body>
</html>
`
