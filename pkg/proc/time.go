// Link to unexported runtime.nanotime()
// copy from https://github.com/gavv/monotime/blob/master/monotime.go
package proc

import (
	_ "unsafe" // required to use //go:linkname
)

//go:noescape
//go:linkname nanotime runtime.nanotime
func nanotime() int64
