package main

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

type lockCounter struct {
	// 0 -> unlock, 1 -> write lock, 2 -> read lock
	state  int
	lockID map[int]bool
}

var lockMap = map[string]*lockCounter{}
var uid int // uid its incrementing counter
var mu sync.Mutex

func lock(path string) int {
	// log.Println("lock path=", path)
	mu.Lock()
	defer mu.Unlock()

	counter := lockMap[path]
	if counter == nil {
		counter = &lockCounter{lockID: make(map[int]bool)}
		lockMap[path] = counter
	}
	if counter.state == 0 {
		counter.state = 1
		id := uid
		uid++
		counter.lockID[id] = true
		return id
	} else {
		return -1
	}
}

func unlock(path string, lockID int) bool {
	// log.Println("unlock path=", path, ", id=", lockID)
	mu.Lock()
	defer mu.Unlock()

	counter := lockMap[path]
	if counter == nil || counter.state != 1 {
		return false
	}

	if _, ok := counter.lockID[lockID]; !ok {
		return false
	}

	delete(counter.lockID, lockID)
	counter.state = 0
	return true
}

func rlock(path string) int {
	// log.Println("rlock path=", path)
	mu.Lock()
	defer mu.Unlock()

	counter := lockMap[path]
	if counter == nil {
		counter = &lockCounter{lockID: make(map[int]bool)}
		lockMap[path] = counter
	}
	if counter.state == 0 || counter.state == 2 {
		counter.state = 2

		id := uid
		uid++
		counter.lockID[id] = true
		// log.Println("rlock path=", path, counter)
		return id
	} else {
		return -1
	}
}

func runlock(path string, lockID int) bool {
	// log.Println("runlock path=", path, ", id=", lockID, lockMap[path])
	mu.Lock()
	defer mu.Unlock()

	counter := lockMap[path]
	if counter == nil || counter.state != 2 {
		return false
	}

	if _, ok := counter.lockID[lockID]; !ok {
		return false
	}
	delete(counter.lockID, lockID)

	if len(counter.lockID) == 0 {
		counter.state = 0
	}
	return true
}

func lHandler(w http.ResponseWriter, r *http.Request, readLock bool) {
	query := r.URL.Query()
	if _, ok := query["key"]; !ok {
		fmt.Fprintf(w, "failure\n")
		return
	}
	path := r.URL.Query().Get("key")
	lockID := -1
	if readLock {
		lockID = rlock(path)
	} else {
		lockID = lock(path)
	}

	if lockID == -1 {
		fmt.Fprintf(w, "retry\n")
	} else {
		fmt.Fprintf(w, strconv.Itoa(lockID)+"\n")
	}
}

func ulHandler(w http.ResponseWriter, r *http.Request, readUnLock bool) {
	query := r.URL.Query()
	if _, ok := query["key"]; !ok {
		fmt.Fprintf(w, "failure\n")
		return
	}
	if _, ok := query["lock-id"]; !ok {
		fmt.Fprintf(w, "failure\n")
		return
	}

	path := r.URL.Query().Get("key")
	stringID := r.URL.Query().Get("lock-id")
	if len(stringID) == 0 {
		fmt.Fprintf(w, "failure\n")
		return
	}

	lockID, err := strconv.Atoi(stringID)
	if err != nil {
		fmt.Println(err)
		return
	}
	res := false
	if readUnLock {
		res = runlock(path, lockID)
	} else {
		res = unlock(path, lockID)
	}

	if res {
		fmt.Fprintf(w, "success\n")
	} else {
		fmt.Fprintf(w, "failure\n")
	}
}

func lockHandler(w http.ResponseWriter, r *http.Request) {
	lHandler(w, r, false)
}

func unlockHandler(w http.ResponseWriter, r *http.Request) {
	ulHandler(w, r, false)
}

func rlockHandler(w http.ResponseWriter, r *http.Request) {
	lHandler(w, r, true)
}

func runlockHandler(w http.ResponseWriter, r *http.Request) {
	ulHandler(w, r, true)
}

// The four REST APIs will look like this:
// POST http://IP:port/lock?key=PATH
// POST http://IP:port/unlock?key=PATH&lock-id=lockID
// POST http://IP:port/rlock?key=PATH
// POST http://IP:port/runlock?key=PATH&lock-id=lockID
func main() {
	uid = 1
	http.HandleFunc("/lock", lockHandler)
	http.HandleFunc("/unlock", unlockHandler)
	http.HandleFunc("/rlock", rlockHandler)
	http.HandleFunc("/runlock", runlockHandler)

	http.ListenAndServe(":8090", nil)
}
