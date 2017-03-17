package wsworld

/*

   Summary of the comms protocol:
   Each message has a 1 char type indicator, followed by the blobs of the message.
   The type indicator and the blobs are separated from each other by 0x1e (30).
   The fields inside a blob are separated by 0x1f (31).

   v (30) p (31) #ffffff (31) 25.2 (31) 54.7 (31) 2.2 (31) 1.3 (30) p (31) #ff0000 (31) 127.4 (31) 339.7 (31) -1.0 (31) 0.4
   |   |                                                         |
   |   |  ---------------------- blob ------------------------   |  ------------------------- blob ------------------------
   |   |                                                         |
   | recsep                                                    recsep
   |
  type

*/

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
            "var %s = new Image();\n%s.src = \"http://%s%s%s\";",
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
"use strict";

{{.ImageLoaders}}

function start_wsworld_client() {

    var that = {};

    var WIDTH = {{.Width}};
    var HEIGHT = {{.Height}};
    var channel_max = 8;
    var canvas = document.querySelector("canvas");
    var virtue = canvas.getContext("2d");

    document.querySelector("canvas").width = {{.Width}};
    document.querySelector("canvas").height = {{.Height}};

    that.ws_frames = 0;
    that.total_draws = 0;
    that.second_last_frame_time = Date.now() - 16;
    that.last_frame_time = Date.now();
    that.all_things = [];

    that.ws = new WebSocket("ws://{{.Server}}{{.WsPath}}");
    that.ws_ready = false;

    that.ws.onopen = function () {
        that.ws_ready = true;
        requestAnimationFrame(that.animate);
    };

    that.ws.onmessage = function (evt) {

        var stuff = evt.data.split(String.fromCharCode(30));    // Our fields are split by ASCII 30 (record sep)
        var frame_type = stuff[0];

        var n;
        var len = stuff.length;

        if (frame_type === "v") {

            // Deal with visual frames.................................................................

            that.all_things.length = 0;         // Clear our list of drawables.

            that.ws_frames += 1;
            that.second_last_frame_time = that.last_frame_time;
            that.last_frame_time = Date.now();

            // Cache the functions to cut down on indirection. Might offer a speedup? Who knows.

            var parse_point_or_sprite = that.parse_point_or_sprite;
            var parse_line = that.parse_line;
            var parse_text = that.parse_text;

            for (n = 1; n < len; n += 1) {

                switch (stuff[n].charAt(0)) {

                case "l":
                    parse_line(stuff[n]);
                    break;
                case "p":
                case "s":
                    parse_point_or_sprite(stuff[n]);
                    break;
                case "t":
                    parse_text(stuff[n]);
                    break;
                }
            }

        } else if (frame_type === "a") {

            // Deal with audio events..................................................................

            for (n = 1; n < len; n += 1) {
                that.play_multi_sound(stuff[n]);
            }

        } else if (frame_type === "d") {

            // Debug messages..........................................................................

            if (len > 0) {
                that.display_debug_message(stuff[1]);
            }
        }
    };

    // Setup keyboard and mouse...

    document.addEventListener("keydown", function (evt) {
        if (that.ws_ready) {
            if (evt.key === " ") {
                that.ws.send("keydown space");
            } else {
                that.ws.send("keydown " + evt.key);
            }
        }
    });

    document.addEventListener("keyup", function (evt) {
        if (that.ws_ready) {
            if (evt.key === " ") {
                that.ws.send("keyup space");
            } else {
                that.ws.send("keyup " + evt.key);
            }
        }
    });

    canvas.addEventListener("mousedown", function (evt) {
        var x = evt.clientX - canvas.offsetLeft;
        var y = evt.clientY - canvas.offsetTop;
        that.ws.send("click " + evt.button.toString() + " " + x.toString() + " " + y.toString());
    });

    that.parse_point_or_sprite = function (blob) {

        var elements = blob.split(String.fromCharCode(31));

        var thing = {};

        thing.type = elements[0];

        if (thing.type === "p") {
            thing.colour = elements[1];
        } else if (thing.type === "s") {
            thing.varname = elements[1];
        }

        thing.x = parseFloat(elements[2]);
        thing.y = parseFloat(elements[3]);
        thing.speedx = parseFloat(elements[4]);
        thing.speedy = parseFloat(elements[5]);

        that.all_things.push(thing);
    };

    that.parse_line = function (blob) {

        var elements = blob.split(String.fromCharCode(31));

        var thing = {};

        thing.type = elements[0];
        thing.colour = elements[1];
        thing.x1 = parseFloat(elements[2]);
        thing.y1 = parseFloat(elements[3]);
        thing.x2 = parseFloat(elements[4]);
        thing.y2 = parseFloat(elements[5]);
        thing.speedx = parseFloat(elements[6]);
        thing.speedy = parseFloat(elements[7]);

        that.all_things.push(thing);
    };

    that.parse_text = function (blob) {

        var elements = blob.split(String.fromCharCode(31));

        if (elements.length < 9) {
            return;
        }

        var thing = {};

        thing.type = elements[0];
        thing.colour = elements[1];
        thing.size =  parseFloat(elements[2]);
        thing.font = elements[3];
        thing.x = parseFloat(elements[4]);
        thing.y = parseFloat(elements[5]);
        thing.speedx = parseFloat(elements[6]);
        thing.speedy = parseFloat(elements[7]);
        thing.text = elements[8];

        that.all_things.push(thing);
    };

    that.draw_text = function(t, time_offset) {
        var x = Math.floor(t.x + t.speedx * time_offset / 1000);
        var y = Math.floor(t.y + t.size / 2 + t.speedy * time_offset / 1000);

        virtue.fillStyle = t.colour;
        virtue.textAlign = "center";
        virtue.font = t.size.toString() + "px " + t.font;
        virtue.fillText(t.text, x, y);
    };

    that.draw_point = function (p, time_offset) {
        var x = Math.floor(p.x + p.speedx * time_offset / 1000);
        var y = Math.floor(p.y + p.speedy * time_offset / 1000);
        virtue.fillStyle = p.colour;
        virtue.fillRect(x, y, 1, 1);
    };

    that.draw_sprite = function (sp, time_offset) {
        var x = sp.x + sp.speedx * time_offset / 1000;
        var y = sp.y + sp.speedy * time_offset / 1000;
        virtue.drawImage(window[sp.varname], x - window[sp.varname].width / 2, y - window[sp.varname].height / 2);
    };

    that.draw_line = function (li, time_offset) {
        var x1 = li.x1 + li.speedx * time_offset / 1000;
        var y1 = li.y1 + li.speedy * time_offset / 1000;
        var x2 = li.x2 + li.speedx * time_offset / 1000;
        var y2 = li.y2 + li.speedy * time_offset / 1000;

        virtue.strokeStyle = li.colour;
        virtue.beginPath();
        virtue.moveTo(x1, y1);
        virtue.lineTo(x2, y2);
        virtue.stroke();
    };

    that.draw = function () {

        virtue.clearRect(0, 0, WIDTH, HEIGHT);     // The best way to clear the canvas??

        // As a relatively simple way of dealing with arbitrary timings of incoming data, we
        // always try to draw the object "where it is now" taking into account how long it's
        // been since we received info about it. This is done with this "time_offset" var.

        var time_offset = Date.now() - that.last_frame_time;

        // Cache various things for speed reasons...

        var all_things = that.all_things;
        var len = all_things.length;

        var draw_point = that.draw_point;
        var draw_sprite = that.draw_sprite;
        var draw_line = that.draw_line;
        var draw_text = that.draw_text;

        var n;
        for (n = 0; n < len; n += 1) {

            switch (all_things[n].type) {
            case "l":
                draw_line(all_things[n], time_offset);
                break;
            case "p":
                draw_point(all_things[n], time_offset);
                break;
            case "s":
                draw_sprite(all_things[n], time_offset);
                break;
            case "t":
                draw_text(all_things[n], time_offset);
                break;
            }
        }
    };

    that.animate = function () {

        if (that.ws_frames > 0) {
            that.total_draws += 1;
        }

        if (that.total_draws % 10 === 0) {
            document.getElementById("ws_frames").innerHTML = that.ws_frames;
            document.getElementById("total_draws").innerHTML = that.total_draws;
        }

        that.draw();
        requestAnimationFrame(that.animate);
    };

    that.display_debug_message = function (s) {
        document.getElementById("debug_msg").innerHTML = "--- " + s;
    };

    // Sound from Thomas Sturm: http://www.storiesinflight.com/html5/audio.html

    that.init_sound = function () {
        that.audiochannels = [];
        while (that.audiochannels.length < channel_max) {
            that.audiochannels.push({channel: new Audio(), finished: -1});
        }
    };

    that.play_multi_sound = function (s) {
        var a;
        var thistime;

        for (a = 0; a < that.audiochannels.length; a += 1) {
            thistime = new Date();
            if (that.audiochannels[a].finished < thistime.getTime()) {
                that.audiochannels[a].finished = thistime.getTime() + document.getElementById(s).duration * 1000;
                that.audiochannels[a].channel.src = document.getElementById(s).src;
                that.audiochannels[a].channel.load();
                that.audiochannels[a].channel.play();
                break;
            }
        }
    };

    that.init_sound();
    return that;
}

start_wsworld_client();

</script>

</body>
</html>
`
