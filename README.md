# lockServer
simple read write lock server over http

supported api
POST http://localhost:8090/lock?key=PATH
POST http://localhost:8090/unlock?key=PATH&lock-id=lockID
POST http://localhost:8090/rlock?key=PATH
POST http://localhost:8090/runlock?key=PATH&lock-id=lockID
