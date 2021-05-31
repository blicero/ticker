// Time-stamp: <2021-05-31 09:48:25 krylon>
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
    const year = t.getYear() + 1900;
    const month = fmtDateNumber(t.getMonth() + 1);
    const day = fmtDateNumber(t.getDate());
    const hour = fmtDateNumber(t.getHours());
    const minute = fmtDateNumber(t.getMinutes());
    const second = fmtDateNumber(t.getSeconds());

    const s =
        year + "-" + month + "-" + day +
        " " + hour + ":" + minute + ":" + second;
    return s;
} // function timeStampString(t)

function fmtDuration(seconds) {
    let minutes = 0, hours = 0;

    while (seconds > 3599) {
        hours++;
        seconds -= 3600;
    }

    while (seconds > 59) {
        minutes++;
        seconds -= 60;
    }

    if (hours > 0) {
        return `${hours}h${minutes}m${seconds}s`;
    } else if (minutes > 0) {
        return `${minutes}m${seconds}s`;
    } else {
        return `${seconds}s`;
    }
} // function fmtDuration(seconds)

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
        // intervalField.setInterval(interval); // ???
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
                         var row_id = `#item_${item_id}`;
                         var row = $(row_id);
                         if (new_rating <= 0.0) {
                             content = '<img src="/static/emo_boring.png" />';
                             row.addClass("boring");
                         } else {
                             content = '<img src="/static/emo_interesting.png" />';
                             row.removeClass("boring");
                         }

                         content += `<br /><input
        type="button"
        class="btn btn-secondary"
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
            var row_id = `#item_${item_id}`;
            $(row_id).removeClass("boring");
            console.log("Rating on Item " + item_id + " has been cleared.");
        });

    req.fail(function(reply, status_text, xhr) {
        let msg = "Error unrating Item at " + addr + ": "  + status_text;
        console.log(msg);
        alert(msg);
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

                         var tag = `<a class="item_${item_id}_tag_${tag_id}" href="/tag/${tag_id}">${reply.Name}</a>&nbsp;<img class="item_${item_id}_tag_${tag_id}" src="/static/delete.png" role="button" onclick="untag(${item_id}, ${tag_id});" /> &nbsp; `;

                         div.innerHTML += tag;

                         var opt_id = `#${form_id}_opt_${tag_id}`;
                         $(opt_id).hide();
                     },
                     "json");

    req.fail(function(reply, status_text, xhr) {
        console.log(`Error attaching Tag to Item: ${status_text} // ${reply}`);
    });
} // function attach_tag(form_id, item_id)

function untag(item_id, tag_id) {
    var tag = `#item_${item_id}_tag_${tag_id}`;
    var msg = `Remove tag ${tag_id} from Item ${item_id}`;
    console.log(msg);
    // alert(msg);
    
    var req = $.post("/ajax/tag_link_delete",
                     { Tag: tag_id, Item: item_id },
                     function(reply) {
                         console.log(`Successfully detached Tag ${tag_id} from Item ${item_id}`);

                         var label_id = `.item_${item_id}_tag_${tag_id}`;
                         var labels = $(label_id);

                         labels.each(function() { $(this).remove(); });

                         var sel_id = `tag_menu_item_${item_id}`;
                         var opt_id = `#${sel_id}_opt_${tag_id}`;
                         $(opt_id).show();
                     },
                     "json");

    req.fail(function(reply, status_text, xhr) {
        console.log(`Error attaching Tag to Item: ${status_text} // ${reply}`);
    });
} // function untag(item_id, tag_id)

// "/ajax/read_later_mark"

function read_later_show(item_id) {
    var button_id = `#read_later_button_${item_id}`;
    var form_id = `#read_later_form_${item_id}`;

    $(form_id).show();
    $(button_id).hide();
} // function read_later_show(item_id)

function read_later_reset(item_id) {
    var button_id = `#read_later_button_${item_id}`;
    var form_id = `#read_later_form_${item_id}`;

    $(form_id).hide();
    $(button_id).show();
} // function read_later_reset(item_id)

function read_later_mark(item_id) {
    console.log(`IMPLEMENTME: Mark Item ${item_id} for later reading.`);
    var button_id = `#read_later_button_${item_id}`;
    var form_id = `#read_later_form_${item_id}`;
    var num_id = `#later_deadline_num_${item_id}`;
    var unit_sel_id = `#later_deadline_unit_${item_id}`;
    var note_id = `#later_note_${item_id}`;

    var num = $(num_id)[0].value;
    var unit = $(unit_sel_id)[0].value;

    var deadline = num * unit;

    var now = new Date();
    var due_time = Math.floor(now.getTime() / 1000 + deadline);

    var note = $(note_id)[0].value;

    var req = $.post("/ajax/read_later_mark",
                     {
                         ItemID:        item_id,
                         Note:          note,
                         Deadline:      due_time,
                     },
                     function(reply) {
                         if (!reply.Status) {
                             var errmsg = `Error marking Item for later: ${reply.Message}`;
                             console.log(errmsg);
                             alert(errmsg);
                         } else {
                             $(form_id).hide();
                             $(button_id).hide();
                         }
                     },
                     "json");

    req.fail(function(reply, status_text, xhr) {
        console.log(`Error attaching Tag to Item: ${status_text} // ${reply}`);
    });

    console.log(`Deadline is ${due_time}`);

    $(form_id).hide();
    $(button_id).show();
} // function read_later_mark(item_id)

function read_later_mark_read(item_id, item_title) {
    const checkbox_id = `#later_mark_read_${item_id}`;
    const state = $(checkbox_id)[0].checked;
    const url = `/ajax/read_later_set_read/${item_id}/${state ? 1 : 0}`;

    var req = $.get(url,
                    {},
                    function(reply) {
                        if (!reply.Status) {
                            const errmsg = `Error marking Item as read: ${reply.Message}`;
                            console.log(errmsg);
                            alert(errmsg);
                        } else {
                            // Do something!
                            const rowid = `#item_${item_id}`;
                            if (state) {
                                $(rowid).addClass("read");
                                $(rowid).removeClass("urgent");
                            } else {
                                // Instead of just turning on the urgent class,
                                // we should if the item's deadline has actually
                                // passed. What would be the easiest way of
                                // doing that?
                                const now = new Date();
                                const row_id = `#item_${item_id}`;
                                const cell = $(row_id)[0].children[0];
                                const txt = cell.textContent.trim();
                                const deadline = new Date(txt);

                                $(rowid).removeClass("read");
                                if (deadline <= now) {
                                    $(rowid).addClass("urgent");
                                }
                            }
                        }
                    },
                    "json");

    req.fail(function(reply, status_text, xhr) {
        console.log(`Error marking Item as Read: ${status_text} // ${reply}`);
        $(checkbox_id)[0].checked = !state;
    });
} // function read_later_mark_read(item_id, item_title)

function read_later_toggle_read_entries() {
    const checkbox_id = "#hide_old";
    const state = $(checkbox_id)[0].checked;
    const query = "#read_later_list .read";

    if (state) {
        $(query).hide();
    } else {
        $(query).show();
    }
} // function read_later_toggle_read_entries()

function edit_feed(feed_id) {
    console.log(`IMPLEMENTME: edit_feed(${feed_id})`);
    const form_id = "#feed_form";
    const feed = feeds[feed_id];

    $("#form_url")[0].value = feed.url;
    $("#form_name")[0].value = feed.name;
    $("#form_homepage")[0].value = feed.homepage;
    $("#form_interval")[0].value = feed.interval / 60;
    // $("#form_active")[0].checked = feed.active;
    $("#form_id")[0].value = feed.id;
    $(form_id).show();
} // function edit_feed(feed_id)

function feed_form_submit() {
    console.log("IMPLEMENTME: feed_form_submit()");

    const id = $("#form_id")[0].value;
    const name = $("#form_name")[0].value;
    const url = $("#form_url")[0].value;
    const homepage = $("#form_homepage")[0].value;
    const interval = $("#form_interval")[0].value;
    // const active = $("#form_active")[0].checked;

    var feed = feeds[id];

    var req = $.post("/ajax/feed_update",
                     { "ID": id,
                       "Name": name,
                       "URL": url,
                       "Homepage": homepage,
                       "Interval": interval * 60,
                       // "Active": active,
                     },
                     function(reply) {
                         if (reply.Status) {
                             console.log(`Successfully updated Feed ${name}`);

                             var hp = $(`#homepage_${id}`)[0];
                             hp.href = homepage;
                             hp.innerHTML = name;

                             var lnk = $(`#url_${id}`)[0];
                             lnk.href = url;
                             lnk.innerHTML = url;

                             $(`#interval_${id}`)[0].innerHTML = fmtDuration(interval * 60);

                             $("#feed_form").hide();
                         } else {
                             const msg = `Error updating Feed ${name}: ${reply.Message}`;
                             console.log(msg);
                             alert(msg);
                         }
                     },
                     "json");

    req.fail(function(reply, status_text, xhr) {
        console.log(`Error updating Feed: ${status_text} // ${reply}`);
        $(checkbox_id)[0].checked = !state;
    });
} // function feed_submit()

function feed_form_reset() {
    $("#feed_form")[0].reset();
    $("#feed_form").hide();
} // function feed_form_reset()

function toggle_feed_active(feed_id) {
    const checkbox_id = `#feed_active_${feed_id}`;
    const active = $(checkbox_id)[0].checked;

    var req = $.get(`/ajax/feed_set_active/${feed_id}/${active}`,
                    {},
                    function(reply) {
                        if (!reply.Status) {
                            $(checkbox_id)[0].checked = !active;
                            alert(reply.Message);
                        }
                    },
                    "json");

    req.fail(function(reply, status_text, xhr) {
        console.log(`Error toggling Feed ${feed_id}: ${status_text} // ${reply}`);
        $(checkbox_id)[0].checked = !active;
    });
} // function toggle_feed_active(feed_id)

function display_tag_items(tag_id) {
    const url = `/ajax/items_by_tag/${tag_id}`;

    var req = $.post(url,
                     {},
                     function(reply) {
                         if (reply.Status) {
                             $("#item_div")[0].innerHTML = reply.Message;
                             shrink_images();
                         } else {
                             console.log(reply.Message);
                             alert(reply.Message);
                         }
                     },
                     "json");

    req.fail(function(reply, status_text, xhr) {
        console.log(`Error getting Items: ${status_text} - ${xhr}`);
    });
} // function display_tag_items(tag_id)

// Found here: https://stackoverflow.com/questions/3971841/how-to-resize-images-proportionally-keeping-the-aspect-ratio#14731922
function shrink_img(srcWidth, srcHeight, maxWidth, maxHeight) {
    const = Math.min(maxWidth / srcWidth, maxHeight / srcHeight);

    return { width: srcWidth*ratio, height: srcHeight*ratio };
} // function shrink_img(srcWidth, srcHeight, maxWidth, maxHeight)

function shrink_images() {
    const selector = "table.items img";
    const maxHeight = 300;
    const maxWidth = 300;

    $(selector).each(function() {
        let img = $(this)[0];
        if (img.width > maxWidth || img.height > maxHeight) {
            const size = shrink_img(img.width, img.height, maxWidth, maxHeight);

            img.width = size.width;
            img.height = size.height;
        }
    });
} // function shrink_images()

function load_feed_items(feed_id) {
    const div_id = "#item_div";
    const url = `/ajax/items_by_feed/${feed_id}`;

    var req = $.get(url,
                     {},
                     function(reply) {
                         if (reply.Status) {
                             $("#item_div")[0].innerHTML = reply.Message;
                             shrink_images();
                         } else {
                             console.log(reply.Message);
                             alert(reply.Message);
                         }
                     },
                     "json");

    req.fail(function(reply, status_text, xhr) {
        console.log(`Error getting Items: ${status_text} - ${xhr}`);
    });
} // function load_feed_items(feed_id)

function shutdown_server() {
    const url = "/ajax/shutdown";

    if (!confirm("Shut down server?")) {
        return false;
    }

    var req = $.get(url,
                    { AreYouSure: true, AreYouReallySure: true },
                    function(reply) {
                        if (!reply.Status) {
                            const msg = `Error shutting down Server: ${reply.Message}`;
                            console.log(msg)
                            alert(msg);
                        }
                    },
                    "json");

    req.fail(function(reply, status_text, xhr) {
        const  msg = `Error getting Items: ${status_text} - ${xhr}`;
        console.log(msg);
        alert(msg);
    });
} // function shutdown_server()
