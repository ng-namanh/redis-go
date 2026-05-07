package commands

import "sync"

var mutex sync.Mutex
var cache = map[string]any{}
var lists map[string]list = make(map[string]list)
var streams = make(map[string]*Stream)
