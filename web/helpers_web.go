// /home/krylon/go/src/pepper/web/helpers_web.go
// -*- mode: go; coding: utf-8; -*-
// Created on 04. 09. 2019 by Benjamin Walkenhorst
// (c) 2019 Benjamin Walkenhorst
// Time-stamp: <2021-02-11 20:47:23 krylon>
//
// Helper functions for use by the HTTP request handlers

package web

// func errJSON(msg string) []byte {
// 	var res = fmt.Sprintf(`{ "Status": false, "Message": %q }`,
// 		msg)

// 	return []byte(res)
// } // func errJSON(msg string) []byte

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

// Local Variables:  //
// compile-command: "go generate && go vet && go build -v -p 16 && gometalinter && go test -v" //
// End: //
