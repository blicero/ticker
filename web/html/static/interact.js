// Time-stamp: <2021-02-12 23:41:26 krylon>
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

// function toggleIntroVisible() {
//     var item = $("#intro")[0];
//     if (item.hidden) {
//         item.hidden = false;
//     } else {
//         item.hidden = true;
//     }
// } // function toggleIntroVisible()

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


var isNoteEdit = false;
var noteID = -1;
var noteBody = "";
var markup = "";
const renderUrl = "/ajax/render_note";
const saveNoteUrl = "/ajax/save_note";
const saveTitleUrl = "/ajax/rename_note";
const addLabelUrl = "/ajax/label_create";
const addLinkUrl = "/ajax/label_link_create";
const delLinkUrl = "/ajax/label_link_delete";

/*
The ‘content’ attribute of Window objects is deprecated.  Please use ‘window.top’ instead. interact.js:125:8
Ignoring get or set of property that has [LenientThis] because the “this” object is incorrect. interact.js:125:8

*/

function toggleNoteEdit() {
    var div = $("#note-body")[0];

    noteBody = div.innerHTML;

    var editor = '<table> <tbody> <tr> <td>' +
        '<textarea class="editor" rows="10" cols="60" name="body" id="notebody" oninput="onNoteChanged();">' +
        markup +
    '</textarea></td><td><div id="note_preview">' +
        noteBody +
        '</div></td></tr></tbody></table>'
    ;

    div.innerHTML = editor;
    isNoteEdit = true;
    $("#edit_cancel")[0].disabled = false;
    $("#edit_save")[0].disabled = false;
    $("#edit_button")[0].disabled = true;
} // function toggleNoteEdit()

function cancelNoteEdit() {
    var div = $("#note-body")[0];
    div.innerHTML = noteBody;
    noteBody = "";
    $("#edit_button")[0].disabled = false;
    $("#edit_cancel")[0].disabled = true;
    $("#edit_save")[0].disabled = true;
    isNoteEdit = false;
} // function cancelNoteEdit()

function saveNoteEdit(note) {
    var txt = $("#notebody")[0];
    var formdata = {
        "id": note.ID,
        "title": note.Title,
        "body": txt.value
    };
    var request = $.post(
        saveNoteUrl,
        formdata,
        function(res) {
            if (!res.Status) {
                var previewField = $("#note_preview")[0];

                txt.innerHTML = noteBody;
                previewField.innerHTML = "";
                var errmsg = "Sending markup failed: " + response.Content;
                console.log(errmsg);
                logMsg("ERROR", errmsg);
                return;
            } else {
                var now = new Date();
                markup = txt.value;

                var req = $.post(
                    renderUrl,
                    { "Content": markup },
                    function(r2) {
                        if (r2.Status) {
                            var bodyArea = $("#note-body")[0];
                            bodyArea.innerHTML = r2.Content;
                        } else {
                            alert("Cannot render body to HTML");
                        }
                    },
                    "json"
                );

                isNoteEdit = false;
                noteBody = "";
                $("#mod_ts")[0].innerHTML = timeStampString(now);

                var old_rev_str = $("#revision_field")[0].innerHTML;
                var rev = Number.parseInt(old_rev_str);

                rev++;
                $("#revision_field")[0].innerHTML = rev;
            }

            $("#edit_button")[0].disabled = false;
            $("#edit_cancel")[0].disabled = true;
            $("#edit_save")[0].disabled = true;
            isNoteEdit = false;
            noteBody = "";
        },
        "json"
    ).fail(function() {
            logMsg("ERROR", "Saving Note " + note.ID + " failed.");
        });
} // function saveNoteEdit()

const previewUpdateInterval = 2500;
var updatePreview = false;

function toggle_note_preview() {
    updatePreview = !updatePreview;
} // function toggle_note_preview()

var noteCheckSum = "";
var noteChanged = false;

function onNoteChanged() {
    var editor = $("#notebody")[0];
    var txt = editor.value;

    var cksum = sha512(txt);

    if (cksum == noteCheckSum) {
        console.log("Note has *not* changed?!?!?");
        logMsg("ERROR", "onNoteChanged has been called, but the Note has " +
               "not changed at all")
        noteChanged = false;
        return;
    }

    noteChanged = true;
    noteCheckSum = cksum;
    console.log("Note has changed.");

    if (txt == undefined || txt == "") {
        return;
    }

    var payload = {
        "Content": txt
    };

    var req = $.post(
        renderUrl,
        payload,
        function(res) {
            if (res.Status) {
                var previewPanel = $("#note_preview")[0];

                previewPanel.innerHTML = res.Content
            } else {
                var msg = renderUrl + " failed: " + res.Message;
                console.log(msg);
                alert(msg);
            }
        },
        "json"
    ).fail(function() {
        const msg = "Rendering Note text failed";
        console.log(msg);
        logMsg("ERROR", msg);
    });
} // function noteChanged()

function update_note_preview() {
    try {
        if (updatePreview) {
            var editor = $("#notebody")[0];
            var rawText = editor.value;

            if (rawText == undefined || rawText == "") {
                return;
            }

            var payload = {
                "Content": rawText
            };

            var req = $.post(
                renderUrl,
                payload,
                function(res) {
                    if (res.Status) {
                        var previewPanel = $("#note_preview")[0];

                        previewPanel.innerHTML = res.Content
                    } else {
                        var msg = renderUrl + " failed: " + res.Message;
                        console.log(msg);
                        alert(msg);
                    }
                },
                "json"
            ).fail(function() {
                const msg = "Rendering Note text failed";
                console.log(msg);
                logMsg("ERROR", msg);
            });
        }
    }
    finally {
        window.setTimeout(update_note_preview, previewUpdateInterval);
    }
} // function update_note_preview()

var noteTitle = "";
var noteTitleOld = "";

function title_edit() {
    var div = $("#title")[0];
    noteTitleOld = div.innerHTML;

    console.log("Title: " + '"' + noteTitle + '"');

    div.innerHTML =
        '<input type="text" id="title_editor" value="' +
        noteTitle +
        '" />' +
        '&nbsp;' +
        '<input type="button" value="Save" onclick="title_save();" />' +
        '<input type="button" value="Cancel" onclick="title_edit_cancel();" />';
} // function title_edit(old_title)

function title_save() {
    var newTitle = $("#title_editor")[0].value;

    if (newTitle != noteTitleOld) {
        var formdata = {
            "id": note.ID,
            "title": newTitle
        }

        var request = $.post(
            saveTitleUrl,
            formdata,
            function(res) {
                if (res.Status) {
                    noteTitle = newTitle;
                } else {
                    alert(res.Message);
                }

                var div = $("#title")[0];

                div.innerHTML =
                    '<div onclick="title_edit();">' +
                    noteTitle +
                    '</div>';

                $("#page_title")[0].innerHTML = noteTitle;
                document.title = noteTitle;
            },
            "json"
        ).fail(function() {
                logMsg("ERROR", "Updating title of Note " + note.ID + " failed");
            });
        
    } else {
        $("#title")[0].innerHTML =
            '<div onclick="title_edit();">' +
            noteTitle +
            '</div>';
    }
} // function title_save()

function title_edit_cancel() {
    var div = $("#title")[0];
    div.innerHTML =
        noteTitleOld;
        // '<div onclick="title_edit();">' +
        // noteTitleOld +
        // '</div>';
} // function title_edit_cancel()

function clear_note_form() {
    var titleInput = $("#note_title")[0];
    var bodyInput = $("#notebody")[0];

    titleInput.value = "";
    bodyInput.value = ""
} // function clear_note_form()

const updateColorURL = "/ajax/note_update_color";

function note_color_change(noteID) {
    var color = $("#note_color")[0].value;
    var bod = document.body;

    var req = $.post(
        updateColorURL,
        {
            id: noteID,
            color: color.substring(1),
        },
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                postMessage(new Date(), "ERROR", res.Message);
                color = "#D4D4D4";
                $("#note_color")[0].value = color;
            } else {
                bod.style["background-color"] = color;
            }
        },
        "json"
    ).fail(function() {
        var msg = "Error setting color for Note " + noteID.toString();
        postMessage(new Date(), "ERROR", msg);
        console.log(message);

        color = "#D4D4D4";
        $("#note_color")[0].value = color;
        bod.style["background-color"] = color;
    });
} // function note_color_change()

function note_color_reset(noteID) {
    const color = "#D4D4D4";
    var bod = document.body;

    var req = $.post(
        updateColorURL,
        {
            id: noteID,
            color: color.substring(1),
        },
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                postMessage(new Date(), "ERROR", res.Message);
            } else {
                $("#note_color")[0].value = color;
                bod.style["background-color"] = color;
            }
        },
        "json"
    ).fail(function() {
        var msg = "Error resetting color for Note " + noteID.toString();
        postMessage(new Date(), "ERROR", msg);
        console.log(message);
        // $("#note_color")[0].value = color;
        // bod.style["background-color"] = color;
    });
} // function note_color_reset()

function create_label() {
    var name = $("#label_form")[0].value;
    var formdata = { "name": name };

    var req = $.post(
        addLabelUrl,
        formdata,
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                alert(res.Message);
            }
            else {
                var list = $("#label_list")[0];
                var item =
                    '<li><a href="/label/' +
                    res.LabelID +
                    '">' +
                    name +
                    '</a>' +
                    '&nbsp;' +
                    '<input class="small" type="button" value="Delete" onclick="delete_label(' +
                    res.LabelID +
                    ');" ' +
                    ' /> </li>';

                list.innerHTML += item;
                $("#label_form")[0].value = "";
            }
        },
        "json"
    );
} // function create_label()

function delete_label(id) {
    var url = "/ajax/label_delete/" + id;

    var req = $.get(
        url,
        {},
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                alert(res.Message);
            }
            else {
                var item = $("#label_item_" + id)[0];

                item.remove();
            }
        },
        "json"
    );
} // function delete_label()

function add_label_link(noteID) {
    var labelID = $("#linklist")[0].value;

    console.log("add_label_link(" + noteID + ") to Label " + labelID);

    var formdata = {
        "note": noteID,
        "label": labelID
    };

    var request = $.post(
        addLinkUrl,
        formdata,
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                alert(res.Message);
            }
            else {
                var name = $("#linklist")[0]
                    .selectedOptions[0]
                    .firstChild
                    .textContent;

                console.log("Add link to Label " + name);

                var list = $("#linked_labels")[0];

                var item =
                    '<li><a href="/label/' +
                    labelID +
                    '">' +
                    name +
                    '</a></li>';

                console.log("New list item: " + item);

                list.innerHTML += item;
            }
        },
        "json"
    );
} // function add_label_link()

function delete_label_link(noteID, labelID) {
    console.log("delete_label_link(" + noteID + ', ' + labelID + ") to Label " + labelID);

    var formdata = {
        "note": noteID,
        "label": labelID
    };

    var request = $.post(
        delLinkUrl,
        formdata,
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                //alert(res.Message);
                logMsg("ERROR", res.Message);
            }
            else {
                var selector = "#label_item_" + labelID;
                var item = $(selector)[0];

                console.log(item);

                item.remove();
            }
        },
        "json"
    );
} // function delete_label_link(noteID)


function shutdown_server() {
    const msg = "Are you sure you want to shut down the Server?";
    const shutdown_url = "/ajax/shutdown";
    var response =  confirm(msg);

    /*
$.ajax({
  type: "POST",
  url: url,
  data: data,
  success: success,
  dataType: dataType
});
     */

    if (response) {
        var success_handler = function(response, status, req) {
            alert("Server has been shut down: " + status);
            var view = $("#msgview")[0];

            view.innerHTML = response;
        };

        var error_handler = function(jx, status, msg) {
            alert("Shutdown failed (" + status + "): " + msg);
        }

        // Shut down server
        $.get({
            url: shutdown_url,
            success: success_handler,
            error: error_handler
        }).fail(function() {
                const msg = "Shutting down the server failed";
                console.log(msg);
                //alert(msg);
                logMsg("ERROR", msg);
            });
    }
} // function shutdown_server()

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

function perform_search() {
    var box = $("#search_box")[0];
    var query = box.value;

    document.URL = "/note/all";

    // console.log("User is asking for \"" + query + "\"");
} // function perform_search()


function link_type_form_save() {
    const saveUrl = "/ajax/link_type_save";

    var form_data = {
        "id": $("#id")[0].value,
        "name": $("#lnk_type_name")[0].value,
        "description": $("#lnk_type_desc")[0].value
    };

    var id = Number.parseInt(form_data["id"]);

    var request = $.post(
        saveUrl,
        form_data,
        function(res) {
            if (res.Status) {
                if (id == 0) {
                    var newRow = '<tr id=lnk_type_"' + res.LinkType.ID +
                        '"><td><a href="/link_type/' +
                        res.LinkType.ID + '">' +
                        res.LinkType.Name + '</td><td>' +
                        res.LinkType.Description + '</td></tr>' + "\n";
                    $("#link_type_view")[0].innerHTML += newRow;
                    link_type_form_clear();
                    linkList[res.LinkType.ID] = res.LinkType;
                } else {
                    var rowid = "lnk_type_" + res.LinkType.ID;
                    var row = $("#" + rowid)[0];

                    var content = '<td><a href="/link_type/' +
                        res.LinkType.ID + '">' +
                        res.LinkType.Name +
                        "<hr />" +
                        '<input type="button" class="small" value="Edit" onclick="link_type_form_load(' + id + ');" />' +
                        '</td> <td>' +
                        res.LinkType.Description + '</td>';

                    row.innerHTML = content;
                    link_type_form_clear();
                }
            } else {
                logMsg("ERROR", "Could not save LinkType: " + res.Message);
            }
        },
        "json"
    );
} // function link_type_form_save()

function link_type_form_clear() {
    console.log("Reset Link Type form");
    $("#id")[0].value = "0";
    $("#lnk_type_name")[0].value = "";
    $("#lnk_type_desc")[0].value = "";
} // function link_type_form_clear()

function link_type_form_load(typeID) {
    var item = linkList[typeID];

    $("#id")[0].value = item.ID;
    $("#lnk_type_name")[0].value = item.Name;
    $("#lnk_type_desc")[0].value = item.Description;
} // function link_type_form_load(typeID)

function link_type_delete(linkID) {
    const delURL = "/ajax/link_type_delete/" + linkID;

    if (!confirm("Delete Link Type " + linkID + "?")) {
        return;
    }

    $.post(
        delURL,
        {},
        function(res) {
            if (res.Status) {
                var rowID = "#lnk_type_" + linkID;
                var row = $(rowID)[0];

                row.hidden = true;
            }
        },
        "json"
    )
        .fail(function() {
            var msg = "link_type_delete(" +
                linkID +
                ") failed";
            console.log(msg);
            alert(msg);
        });
} // function link_type_delete(linkID)

function note_link_type_select() {
    const fetchNoteUrl = "/ajax/note_all";
    var linkList = $("#link_type_list")[0];
    var opt = linkList.selectedOptions[0];
    var linkName = opt.innerText;
    var notes;

    var req = $.post(
        fetchNoteUrl,
        {},
        function(res) {
            if (res.Status) {
                var hostSelect = $("#note_link_list")[0];

                var options = new Array(res.Notes.length);

                for (var i = 0; i < res.Notes.length; i++) {
                    options[i] = "<option value=\"" +
                        res.Notes[i].ID +
                        "\">" +
                        res.Notes[i].Title +
                        "</option>";
                }

                var listContent = options.join("\n");

                hostSelect.innerHTML = listContent;
                hostSelect.disabled = false;
            } else {
                console.log(res.Message);
                alert(res.Message);
            }
        },
        "json"
    );
} // function note_link_dst_load()

function note_link_create(srcID) {
    const createURL = "/ajax/link_create";
    var hostMenu = $("#note_link_list")[0];

    if (hostMenu.disabled) {
        return;
    }

    var linkTypeList = $("#link_type_list")[0];
    var noteID = Number.parseInt(hostMenu.selectedOptions[0].value);
    var typeID = Number.parseInt(linkTypeList.selectedOptions[0].value);

    var requestData = {
        "Src": srcID,
        "Dst": noteID,
        "Type": typeID
    };

    var req = $.post(
        createURL,
        requestData,
        function(res) {
            if (res.Status) {
                var txt = '<tr id="link_item_' +
                    res.LinkID +
                    '">\n<td>\n<a href="/link_type_' +
                    res.Type.ID +
                    '">' +
                    res.Type.Name +
                    '</a>\n</td>\n<td><a href="/note/' +
                    res.Note.ID +
                    '">' +
                    res.Note.Title +
                    '</a>\n</td>\n<td>' +
                    '<input type="button" value="Delete" onclick="note_link_delete(' +
                    res.LinkID +
                    ');" />\n</td>\n</tr>\n';

                if (linkCnt++ == 0) {
                    var nodes = $(".linkedNoteList");

                    for (var i = 0; i < nodes.length; i++) {
                        console.log(nodes[i].tagName);
                        nodes[i].hidden = false;
                    }
                    // $(".linkedNoteList").each(function(x) { x.hidden = false; });
                }

                var linkBody = $("#linkedNoteBody")[0];
                linkBody.innerHTML += txt;
            } else {
                //alert(res.Message);
                logMsg("ERROR", res.Message);
            }
        },
        "json"
    ).fail(function() {
        logMsg("ERROR", "Linking Note " + srcID + " to " + noteID + " failed");
    });
} // function note_link_create()

function note_link_delete(id) {
    const deleteURL = "/ajax/link_delete"

    var formData = {
        "linkID": id,
        "dry": false
    };

    var req = $.post(
        deleteURL,
        formData,
        function(res) {
            if (res.Status) {
                var itemID = "link_item_" + id;
                var item = $("#" + itemID)[0];

                if (item != undefined) {
                    item.remove();
                    if (--linkCnt == 0) {
                        hide_empty_link_list();
                    }
                }
            } else {
                console.log(res.Message);
                alert(res.Message);
            }
        },
        "json"
    ).fail(function() {
        logMsg("ERROR", "Deleting Link " + id + " has failed.");
    });
} // function note_link_delete(id)

function note_delete(noteID) {
    const noteDeleteURL = "/ajax/note_delete";

    if (!confirm("Delete Note " + noteID.toString() + "?")) {
        return;
    }

    var req = $.post(
        noteDeleteURL,
        { id: noteID },
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                postMessage(new Date(), "ERROR", res.Message);
            } else {
                var rowID = "#note_" + noteID.toString();
                var row = $(rowID)[0];
                row.hidden = true;
            }
        },
        "json"
    ).fail(function() {
        var msg = "Error deleting Note " + noteID;
        console.log(msg);
        postMessage(new Date(), "ERROR", msg);

    });
} // function note_delete(noteID)

function note_delete_from_view(noteID) {
    const noteDeleteURL = "/ajax/note_delete";

    if (!confirm("Delete Note " + noteID.toString() + "?")) {
        return;
    }

    var req = $.post(
        noteDeleteURL,
        { id: noteID },
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                postMessage(new Date(), "ERROR", res.Message);
            } else {
                window.location = "/note/all";
            }
        },
        "json"
    ).fail(function() {
        var msg = "Error deleting Note " + noteID;
        console.log(msg);
        postMessage(new Date(), "ERROR", msg);

    });
} // function note_delete_from_view(noteID)

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

function toggleRemindersVisible() {
    var display = $("#reminder_tbl")[0];

    display.hidden = !display.hidden;
} // function toggleRemindersVisible()

function fileDelete(id) {
    var msg = "Imeplement me: FileDelete(" + id + ")";
    var deleteURL = "/ajax/file/" + id + "/delete";

    if (!confirm("Delete file #" + id + "?")) {
        return;
    }

    $.post(
        deleteURL,
        {},
        function(res) {
            if (res.Status) {
                var rowid = "#file_det_" + id;
                $(rowid)[0].hidden = true;

                console.log("Test fileDelete");
            }
        },
        "json")
        .fail(function() {
            alert("ERROR");
        })

    console.log(msg);
    logMsg("ERROR", msg);
} // function fileDelete(id)

function fileRename(id) {
    var msg = "Implement me: FileRename(" + id + ")";
    console.log(msg);
    var renameURL = "/ajax/file/" + id + "/rename";
    var newName = prompt("New name: ");
    if (newName == null || newName == "") {
        return;
    }

    if (!confirm("Rename file #" + id + "?")) {
        return;
    }

    $.post(
        renameURL,
        {
            "name": newName
        },
        function(res) {
            if (res.Status) {
                $("#file_det_" + id)[0].innerHTML = newName;
            }
        },
        "json")
        .fail(function() {
            alert("ERROR");
        })

    console.log(msg);
    logMsg("ERROR", msg);
} // function fileRename(id)

function fileCleanup() {
    const cleanURL = "/file/upload/clean";

    if (!confirm("Delete temporary files?")) {
        return;
    }

    var req = $.post(
        cleanURL,
        {},
        function(res) {
            if (!res.Status) {
                console.log(res.Message);
                postMessage(new Date(), "ERROR", res.Message);
            }
        },
        "json"
    ).fail(function() {
        var msg = "Request to " + cleanURL + " failed: " + arguments[1];
        console.log(msg);
        alert(msg);
    });
} // function fileCleanup()

var kwOld = {};

function fileKeywordsEdit(fileID) {
    if (kwOld.hasOwnProperty(fileID)) {
        return;
    }

    var divID = "#file_keywords_" + fileID;
    var div = $(divID)[0];
    var txt = div.innerText;

    var inputID = 'file_keywords_input_' + fileID;
    var divContent =
        '<input type="text" id="' +
        inputID + '"' +
        ' value="' +
        txt + '" ' +
        '/>' +
        '&nbsp;' +
        '<input type="button" value="Save" onclick="fileKeywordsSave(' +
        fileID +
        ');" />' +
        '&nbsp;' +
        '<input type="button" value="Cancel" onclick="fileKeywordsCancel(' +
        fileID +
        ');" />';

    kwOld[fileID] = div.innerHTML;
    div.innerHTML = divContent;
    var input = $("#" + inputID);

    if (input == null || input == undefined || input.length == 0) {
        console.log("Argh!");
        return;
    }

    input[0].value = div.innerText;
} // function fileEditwords(fileID)

function fileKeywordsSave(fileID) {
    var newKeywords = kwOld[fileID];

    try {
        var newKeywords = $("#file_keywords_input_" + inputID)[0].value;

        var divID = "#file_keywords_" + fileID;
        var div = $(divID)[0];

        div.innerHTML = newKeywords;
    }
    finally {
        delete kwOld[fileID];
    }
} // function fileKeywordsSave(fileID, inputID)

function fileKeywordsCancel(fileID) {
    try {
        var divID = "#file_keywords_" + fileID;
        var div = $(divID)[0];
        div.innerHTML = kwOld[fileID];
    }
    finally {
        delete kwOld[fileID];
    }
} // function fileKeywordsCancel(fileID)

function reminder_add(noteID) {
    const reminderAddURL = "/ajax/reminder/add";

    var reminder = {
        "Date": $("#reminder_date")[0].value,
        "Time": $("#reminder_time")[0].value,
        "Title": $("#reminder_title")[0].value,
        "Body": $("#reminder_body")[0].value,
        "NoteID": noteID,
        "Cleared": false,
    };

    if (reminder.Date == "" || reminder.Time == "") {
        return;
    }

    var req = $.post(
        reminderAddURL,
        reminder,
        function(res) {
            if (res.Status) {
                var list = $("#reminder_list")[0];
                var row = '<tr><td>' +
                    reminder.Title +
                    '</td><td>' +
                    reminder.Date + ' ' + reminder.Time +
                    '</td><td>' +
                    reminder.Body +
                    '</td><td>' +
                    '<input type="button" value="Delete" onclick="reminder_delete(' +
                    res.Reminder.ID + ');"' +
                    '</td></tr>';

                list.innerHTML += row;

                $("#reminder_date")[0].value = "";
                $("#reminder_time")[0].value = "";
                $("#reminder_title")[0].value = "";
                $("#reminder_body")[0].value = "";
            } else {
                console.log(res.Message);
                alert(res.Message);
            }
        },
        "json")
        .fail(function() {
            alert("Adding Reminder failed"); 
        });
} // function reminder_add()

var activeReminders = {};

function reminder_delete(reminderID) {
    console.log("reminder_delete(" + reminderID + ");");
    var row = $("#reminder_display_" + reminderID)[0];
    var title = row.children[2].innerText;

    var question = 'Are you sure you want to delete Reminder "' +
        title +
        '"?';

    if (!confirm(question)) {
        return;
    }

    const delUrl = "/ajax/reminder/del";

    var req = $.post(
        delUrl,
        { "id": reminderID },
        function(res) {
            if (!res.Status) {
                var msg = 'Error deleting Reminder #' + id + ': ' +
                    res.Message;
                console.log(msg);
                alert(msg);
            } else {
                if (activeReminders[reminderID]) {
                    var rowID = "#reminder_display_" + reminderID;
                    $(rowID)[0].hidden = true;
                    delete activeReminders[reminderID];
                }
                row.hidden = true;
            }
        },
        "json"
    )
        .fail(function() {
            var msg = 'Error deleting Reminder "' + title + '"';
            console.log(msg);
            alert(msg);
        });
} // function reminder_delete(reminderID)

function reminder_display(r) {
    const bodyID = "#reminder_list_body";

    if (activeReminders[r.ID]) {
        return;
    }

    var body = $(bodyID)[0];
    if (body != undefined) {
        var row = '<tr id="reminder_display_' +
            r.ID + '"><td><a href="/reminder/edit/' + r.ID + '">' + 
            r.Title + '</a></td><td>' +
            r.Timestamp + '</td><td>' +
            r.Body + '</td><td>' +
            '<input type="button" value="Delete" onclick="reminder_delete(' +
            r.ID + ')" /></td></tr>';
        body.innerHTML += row;
        activeReminders[r.ID] = true;
    }
} // function reminder_display(r)

// var fetchPending = false;

// function reminder_get_pending() {
//     const remURL = "/ajax/reminder/pending";

//     try {
//         if (fetchPending) {
//             var req = $.get(
//                 remURL,
//                 {},
//                 function(res) {
//                     if (!res.Status) {
//                         console.log(res.Message);
//                         postMessage(new Date(), "ERROR", res.Message);
//                     } else {
//                         for (var i = 0; i < res.Reminders.length; i++) {
//                             var r = res.Reminders[i];
//                             reminder_display(r);
//                         }
//                     }
//                 },
//                 "json"
//             )
//                 .fail(function() {
//                     var msg = "Error querying pending Reminders: " + arguments[1];
//                     console.log(msg)
//                     postMessage(new Date(), "ERROR", msg);
//                 });
//         }
//     }
//     finally {
//         window.setTimeout(reminder_get_pending, 1000);
//     }
// } // function reminder_get_pending()
