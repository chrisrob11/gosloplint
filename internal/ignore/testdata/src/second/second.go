// Package second is read by plain, the second copy, which leaves names of
// no known analyzer to the first.
package second

//gosloplint:ignore plain:x plain has no checks // want `ignore: //gosloplint:ignore plain:x: plain has no checks to name; write //gosloplint:ignore plain <reason>`
var plainBadCheck int // want `plain: plainBadCheck`

//gosloplint:ignore plain the plain analyzer's own finding
var plainQuiet int

//gosloplint:ignore nosuch the first copy reports this, not plain

//gosloplint:ignore

//gosloplint:ignore fake fake does not run here

// want +1 `ignore: //gosloplint:ignore plain suppresses no plain finding: delete it`
//gosloplint:ignore plain stale
