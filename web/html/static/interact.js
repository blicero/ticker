// Time-stamp: <2021-02-26 17:57:59 krylon>
// -*- mode: javascript; coding: utf-8; -*-
// Copyright 2015-2020 Benjamin Walkenhorst <krylon@gmx.net>
//
// This file has grown quite a bit larger than I had anticipated.
// It is not a /big/ problem right now, but in the long run, I will have to
// break this thing up into several smaller files.

"use strict";

function defined(x) {
    return undefined != x && null != x;
}

function fmtDateNumber(n) {
    if (n < 10) {
        return "0" + n.toString();
    } else {
        return n.toString();
    }
} // function fmtDateNumber(n)

function timeStampString(t) {
    if (typeof(t) == "string") {
        return t;
    }

    // (1900 + d.getYear()) + "-" + d.getMonth() + "-" + d.getDate() + " " + d.getHours() + ":" + d.getMinutes() + ":" + d.getSeconds()
    var year = t.getYear() + 1900;
    var month = fmtDateNumber(t.getMonth() + 1);
    var day = fmtDateNumber(t.getDate());
    var hour = fmtDateNumber(t.getHours());
    var minute = fmtDateNumber(t.getMinutes());
    var second = fmtDateNumber(t.getSeconds());

    var s =
        year + "-" + month + "-" + day +
        " " + hour + ":" + minute + ":" + second;
    return s;
} // function timeStampString(t)

function beaconLoop() {
    try {
        if (settings.beacon.active) {
            var req = $.get("/ajax/beacon",
                            {},
                            function(response) {
                                var status = "";

                                if (response.Status) {
                                    status =
                                        response.Message +
                                        " running on " +
                                        response.Hostname + 
                                        " is alive at " +
                                        response.Timestamp;
                                } else {
                                    status = "Server is not responding";
                                }

                                var beaconDiv = $("#beacon")[0];

                                if (defined(beaconDiv)) {
                                    beaconDiv.innerHTML = status;
                                    beaconDiv.classList.remove("error");
                                } else {
                                    console.log("Beacon field was not found");
                                }
                            },
                            "json"
                           ).fail(function() {
                               var beaconDiv = $("#beacon")[0];
                               beaconDiv.innerHTML = "Server is not responding";
                               beaconDiv.classList.add("error");
                               //logMsg("ERROR", "Server is not responding");
                           });
        }
    }
    finally {
        window.setTimeout(beaconLoop, settings.beacon.interval);
    }
} // function beaconLoop()

function beaconToggle() {
    settings.beacon.active = !settings.beacon.active;
    saveSetting("beacon", "active", settings.beacon.active);

    if (!settings.beacon.active) {
        var beaconDiv = $("#beacon")[0];
        beaconDiv.innerHTML = "Beacon is suspended";
        beaconDiv.classList.remove("error");
    }

} // function beaconToggle()


/*
The ‘content’ attribute of Window objects is deprecated.  Please use ‘window.top’ instead. interact.js:125:8
Ignoring get or set of property that has [LenientThis] because the “this” object is incorrect. interact.js:125:8

*/

function db_maintenance() {
    const maintURL = "/ajax/db_maint";

    var req = $.get(
        maintURL,
        {},
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                postMessage(new Date(), "ERROR", res.Message);
            } else {
                const msg = "Database Maintenance performed without errors";
                console.log(msg);
                postMessage(new Date(), "INFO", msg);
            }
        },
        "json"
    ).fail(function() {
        var msg = "Error performing DB maintenance";
        console.log(msg);
        postMessage(new Date(), "ERROR", msg);
    });
} // function db_maintenance()

function msgCheckSum(timestamp, level, msg) {
    var line = [ timeStampString(timestamp), level, msg ].join("##");

    var cksum = sha512(line);
    return cksum;
}

var curMessageCnt = 0;

function post_test_msg() {
    var user = $("#msgTestText")[0];
    var msg = user.value;
    var now = new Date();

    postMessage(now, "DEBUG", msg);
} // function post_tst_msg()

function postMessage(timestamp, level, msg) {
    var row = '<tr id="msg_' +
        msgCheckSum(timestamp, level, msg) +
        '"><td>' +
        timeStampString(timestamp) +
        '</td><td>' +
        level +
        '</td><td>' +
        msg +
        '</td></tr>\n';

    msgRowAdd(row);
} // function postMessage(timestamp, level, msg)

function adjustMsgMaxCnt() {
    var cntField = $("#max_msg_cnt")[0];
    var newMax = cntField.valueAsNumber;

    if (newMax < curMessageCnt) {
        var rows = $("#msg_body")[0].children;

        while (rows.length > newMax) {
            rows[rows.length - 1].remove();
            curMessageCnt--;
        }

    }
    
    saveSetting("messages", "maxShow", newMax);
} // function adjustMaxMsgCnt()

function adjustMsgCheckInterval() {
    var intervalField = $("#msg_check_interval")[0];
    if (intervalField.checkValidity()) {
        var interval = intervalField.valueAsNumber;
        intervalField.setInterval(interval);
        saveSetting("messages", "interval", interval);
    }
} // function adjustMsgCheckInterval()

function toggleCheckMessages() {
    var box = $("#msg_check_switch")[0];
    var newVal = box.checked;

    saveSetting("messages", "queryEnabled", newVal);
} // function toggleCheckMessages()

function getNewMessages() {
    const msgURL = "/ajax/get_messages";

    try {
        if (!settings.messages.queryEnabled) {
            return;
        }
        
        var req = $.get(
            msgURL,
            {},
            function(res) {
                if (!res.Status) {
                    var msg = msgURL +
                        " failed: " +
                        res.Message;

                    console.log(msg)
                    alert(msg);
                } else {
                    for (var i = 0; i < res.Messages.length; i++) {
                        var item = res.Messages[i];
                        var rowid =
                            "msg_" +
                            msgCheckSum(item.Time, item.Level, item.Message);
                        var row = '<tr id="' +
                            rowid +
                            '"><td>' +
                            item.Time +
                            '</td><td>' +
                            item.Level +
                            '</td><td>' +
                            item.Message +
                            '</td><td>' +
                            '<input type="button" value="Delete" onclick="msgRowDelete(\'' +
                            rowid +
                            '\');" />' +
                            '</td></tr>\n';

                        msgRowAdd(row);
                    }
                }
            },
            "json"
        );
    }
    finally {
        window.setTimeout(getNewMessages, settings.messages.interval);
    }

} // function getNewMessages()

function logMsg(level, msg) {
    var timestamp = timeStampString(new Date());
    var rowID = "msg_" + sha512(msgCheckSum(timestamp, level, msg));
    var row = '<tr id="' +
        rowID +
        '"><td>' +
        timestamp +
        '</td><td>' +
        level +
        '</td><td>' +
        msg +
        '</td><td>' +
        '<input type="button" value="Delete" onclick="msgRowDelete(\'' +
        rowID +
        '\');" />' +
        '</td></tr>\n';

    $("#msg_display_tbl")[0].innerHTML += row;
} // function logMsg(level, msg)

function msgRowAdd(row) {
    var msgBody = $("#msg_body")[0];

    msgBody.innerHTML = row + msgBody.innerHTML;

    if (++curMessageCnt > settings.messages.maxShow) {
        msgBody.children[msgBody.children.length - 1].remove();
    }

    var tbl = $("#msg_tbl")[0];
    if (tbl.hidden) {
        tbl.hidden = false;
    }
} // function msgRowAdd(row)

function msgRowDelete(rowID) {
    var row = $("#" + rowID)[0];

    if (row != undefined) {
        row.remove();
        if (--curMessageCnt == 0) {
            var tbl = $("#msg_tbl")[0];
            tbl.hidden = true;
        }
    }
} // function msgRowDelete(rowID)

function msgRowDeleteAll() {
    var msgBody = $("#msg_body")[0];
    msgBody.innerHTML = '';
    curMessageCnt = 0;

    var tbl = $("#msg_tbl")[0];
    tbl.hidden = true;
} // function msgRowDeleteAll()

function requestTestMessages() {
    const urlRoot = "/ajax/rnd_message/";

    var cnt = $("#msg_cnt")[0].valueAsNumber;
    var rounds = $("#msg_round_cnt")[0].valueAsNumber;
    var delay = $("#msg_round_delay")[0].valueAsNumber;

    if (cnt == 0) {
        console.log("Generate *0* messages? Alrighty then...");
        return;
    }

    var reqURL = urlRoot + cnt;

    $.get(
        reqURL,
        {
            "Rounds": rounds,
            "Delay": delay,
        },
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                alert(res.Message);
            }
        },
        "json"
    ).fail(function() {
            const msg = "Requesting test messages failed.";
            console.log(msg);
            //alert(msg);
            logMsg("ERROR", msg);
        });
} // function requestTestMessages()

function toggleMsgTestDisplayVisible() {
    var tbl = $("#test_msg_cfg")[0];

    if (tbl.hidden) {
        tbl.hidden = false;

        var checkbox = $("#msg_check_switch")[0];
        settings.messages.queryEnabled = checkbox.checked;
    } else {
        settings.messages.queryEnabled = false;
        tbl.hidden = true;
    }
} // function toggleMsgTmpDisplayVisible()

function toggleMsgDisplayVisible() {
    var display = $("#msg_display_div")[0];

    display.hidden = !display.hidden;
} // function toggleMsgDisplayVisible()

function rate_item(item_id, new_rating) {
    var req = $.post("/ajax/rate_item",
                     { ID: item_id, Rating: new_rating },
                     function(reply) {
                         var content = "";
                         if (new_rating <= 0.0) {
                             content = '<img src="/static/emo_boring.png" />';
                         } else {
                             content = '<img src="/static/emo_interesting.png" />';
                         }

                         content += `<br /><input
        type="button"
        value="Unvote"
        onclick="unvote_item(${item_id});" />`;

                         $("#item_rating_" + item_id)[0].innerHTML = content;
                     },
                     "json");

    req.fail(function(reply, status_text, xhr) {
        console.log("Our Ajax request failed: " + status_text);
        var data = reply; // $.parseJSON(reply.responseText);
        if (data.Status) {
            var msg = "Error rating item - but Status is true?!?!?!";
            alert(msg);
            console.log(msg);
        }
        else {
            var msg = "Error rating item - " + data.Message;
            alert(msg);
            console.log(msg);
        }
    });
} // function rate_item(item_id, new_rating)

function unvote_item(item_id) {
    const addr = "/ajax/unrate_item/" + item_id;
    var req = $.get(
        addr,
        {},
        function(reply) {
            // Display zee buttons!
            alert("Rating on Item " + item_id + " has been cleared.");
        });

    req.fail(function(reply, status_text, xhr) {
        console.log("Error unrating Item at " + addr + ": "  + status_text);
    });
} // function unvote_item(item_id)

function hide_boring_items() {
    console.log("Hiding boring items.");
    $.each($("tr.boring"), function() { $(this).hide(); } );
} // function hide_boring_items()

function show_boring_items() {
    console.log("Displaying boring items.");
    $.each($("tr.boring"), function() { $(this).show(); } );
} // function show_boring_items()

function toggle_hide_boring() {
    console.log("toggle_hide_boring()");

    settings.items.hideboring = !settings.items.hideboring;
    saveSetting("items", "hideboring", settings.items.hideboring);

    if (settings.items.hideboring) {
        hide_boring_items();
    } else {
        show_boring_items();
    }

    return true;
} // function toggle_hide_boring()

function rebuildFTS() {
    var req = $.get("/ajax/rebuild_fts",
                    "",
                    function(reply) {
                        console.log("FTS index has been rebuilt.");
                    });

    req.fail(function(reply, status_text, xhr) {
        var msg = reply + " -- " + status_text;
        console.log(msg);
        alert(msg);
    });
} // function rebuildFTS()

function attach_tag(form_id, item_id) {
    var sel = $("#" + form_id)[0].value;
    var tag_id = parseInt(sel);

    console.log("Attach Tag #" + sel + " to Item " + item_id + ".");

    var req = $.post("/ajax/tag_link_create",
                     { Tag: tag_id, Item: item_id },
                     function(reply) {
                         console.log(`Successfully attached Tag ${sel} to Item ${item_id}`);
                         var div_id = `#tags_${item_id}`;
                         var div = $(div_id)[0];

                         var tag = `<a href="/tag/${tag_id}">${reply.Name}</a>&nbsp;`;

                         div.innerHTML += tag;

                         var opt_id = `#${form_id}_opt_${tag_id}`;
                         var opt = $(opt_id)[0];

                         opt.remove();
                     },
                     "json");

    req.fail(function(reply, status_text, xhr) {
        console.log(`Error attaching Tag to Item: ${status_text} // ${reply}`);
    });
} // function attach_tag(form_id, item_id)
