// Time-stamp: <2021-06-28 20:39:27 krylon>

"use strict"

function sayHello (itemID) {
    const msg = 'Hello there'
    console.log(msg)
    $(itemID)[0].innerText = msg
} // function sayHello (itemID)


