// /home/krylon/go/src/pepper/web/helpers_web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 09. 2019 by Benjamin Walkenhorst
// (c) 2019 Benjamin Walkenhorst
// Time-stamp: <2021-07-19 17:00:02 krylon>
//
// Helper functions for use by the HTTP request handlers

package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"ticker/common"
	"ticker/feed"
)

func errJSON(msg string) []byte {
	var res = fmt.Sprintf(`{ "Status": false, "Message": %q }`,
		jsonEscape(msg))

	return []byte(res)
} // func errJSON(msg string) []byte

func jsonEscape(i string) string {
	b, err := json.Marshal(i)
	if err != nil {
		panic(err)
	}
	// Trim the beginning and trailing " character
	return string(b[1 : len(b)-1])
}

type itemList []feed.Item

func (il itemList) Len() int           { return len(il) }
func (il itemList) Less(i, j int) bool { return il[i].Timestamp.After(il[j].Timestamp) }
func (il itemList) Swap(i, j int)      { il[i], il[j] = il[j], il[i] }

// func getMimeType(path string) (string, error) {
// 	var (
// 		fh      *os.File
// 		err     error
// 		buffer  [512]byte
// 		byteCnt int
// 	)

// 	if fh, err = os.Open(path); err != nil {
// 		return "", err
// 	}

// 	defer fh.Close() // nolint: errcheck

// 	if byteCnt, err = fh.Read(buffer[:]); err != nil {
// 		return "", fmt.Errorf("cannot read from %s: %s",
// 			path,
// 			err.Error())
// 	}

// 	return http.DetectContentType(buffer[:byteCnt]), nil
// }
// func getMimeType(path string) (string, error)

func (srv *Server) baseData(title string, r *http.Request) tmplDataBase {
	return tmplDataBase{
		Title:      title,
		Debug:      common.Debug,
		URL:        r.URL.String(),
		TrainStamp: srv.trainStamp(),
	}
} // func (srv *Server) baseData(title string, r *http.Request) tmplDataBase

// Local Variables:  //
// compile-command: "go generate && go vet && go build -v -p 16 && mygolint ticker/web && go test -v" //
// End: //
