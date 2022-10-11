// /home/krylon/go/src/ticker/web/ajax_data.go
// -*- mode: go; coding: utf-8; -*-
// Created on 19. 06. 2021 by Benjamin Walkenhorst
// (c) 2021 Benjamin Walkenhorst
// Time-stamp: <2022-10-10 22:56:23 krylon>

package web

// go : generate ffjson ajax_data.go

// Types for AJAX responses

// nolint: unused
type ajaxResponse struct {
	Status  bool
	Message string
}

// type ajaxResponseHTML struct {
// 	Status  bool
// 	Message string
// 	HTML    string
// }
