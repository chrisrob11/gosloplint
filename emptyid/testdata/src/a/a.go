package a

import (
	"errors"
	"fmt"
	"log/slog"
)

// Reported: the guard's body leaves without a word.

func returnNil(accountID string) error {
	if accountID == "" { // want `guard on empty accountID exits silently \(return nil\)`
		return nil
	}
	return nil
}

func continues(ids []string) {
	for _, id := range ids {
		if id == "" { // want `guard on empty id exits silently \(continue\)`
			continue
		}
		_ = id
	}
}

func bare(sessionKey string) {
	if sessionKey == "" { // want `guard on empty sessionKey exits silently \(return\)`
		return
	}
	_ = sessionKey
}

type S struct{ UserID string }

func selector(s S) error {
	if s.UserID == "" { // want `guard on empty s.UserID exits silently`
		return nil
	}
	return nil
}

func reversed(orderID string) error {
	if "" == orderID { // want `guard on empty orderID exits silently`
		return nil
	}
	return nil
}

func multiNil(orderID string) (*int, error) {
	if orderID == "" { // want `guard on empty orderID exits silently \(return nil\)`
		return nil, nil
	}
	return nil, nil
}

// A no-op before the exit does not make the guard loud.
func countsThenReturns(orderID string, n *int) error {
	if orderID == "" { // want `guard on empty orderID exits silently`
		*n++
		return nil
	}
	return nil
}

// A named empty constant is the same test.
const none = ""

func namedConstant(userID string) error {
	if userID == none { // want `guard on empty userID exits silently`
		return nil
	}
	return nil
}

// The default suffixes include name.
func byName(name string) error {
	if name == "" { // want `guard on empty name exits silently`
		return nil
	}
	return nil
}

// Snake case splits the same way.
func snake(order_id string) error {
	if order_id == "" { // want `guard on empty order_id exits silently`
		return nil
	}
	return nil
}

// Compound guards: the emptiness test rides along with another.

func orAfterOK(tup []any) {
	for _, v := range tup {
		orderID, ok := v.(string)
		if !ok || orderID == "" { // want `guard on empty orderID exits silently \(continue\)`
			continue
		}
		_ = orderID
	}
}

func orBeforeOK(userID string, ok bool) error {
	if userID == "" || !ok { // want `guard on empty userID exits silently`
		return nil
	}
	return nil
}

func andJoined(accountID string, ok bool) error {
	if ok && accountID == "" { // want `guard on empty accountID exits silently`
		return nil
	}
	return nil
}

func nestedOr(a, b bool, orderID string) error {
	if a || b || orderID == "" { // want `guard on empty orderID exits silently`
		return nil
	}
	return nil
}

func parenthesised(ok bool, userID string) error {
	if !ok || (userID == "") { // want `guard on empty userID exits silently`
		return nil
	}
	return nil
}

// One guard is one finding, even when both operands are identifiers.
func twoIdents(userID, orderID string) error {
	if userID == "" || orderID == "" { // want `guard on empty userID exits silently`
		return nil
	}
	return nil
}

// len(x) == 0 is the same test on a slice or a string.

func lenZero(leaseID []byte) error {
	if len(leaseID) == 0 { // want `guard on empty leaseID exits silently`
		return nil
	}
	return nil
}

func zeroLen(leaseID []byte) error {
	if 0 == len(leaseID) { // want `guard on empty leaseID exits silently`
		return nil
	}
	return nil
}

func lenSelector(c struct{ lastKey []byte }) error {
	if len(c.lastKey) == 0 { // want `guard on empty c.lastKey exits silently`
		return nil
	}
	return nil
}

func lenInOr(ok bool, ids []string) error {
	if !ok || len(ids) == 0 { // want `guard on empty ids exits silently`
		return nil
	}
	return nil
}

func lenPlural(userIDs []string) error {
	if len(userIDs) == 0 { // want `guard on empty userIDs exits silently`
		return nil
	}
	return nil
}

func sessionID() string { return "" }

func lenCall() error {
	if len(sessionID()) == 0 { // want `guard on empty sessionID exits silently`
		return nil
	}
	return nil
}

// The switch spelling.

func switchReturn(userID string) error {
	switch userID {
	case "": // want `guard on empty userID exits silently \(return nil\)`
		return nil
	}
	return nil
}

func switchContinue(ids []string) {
	for _, recordID := range ids {
		switch recordID {
		case "": // want `guard on empty recordID exits silently \(continue\)`
			continue
		}
		_ = recordID
	}
}

func switchBare(s S) {
	switch s.UserID {
	case "": // want `guard on empty s.UserID exits silently \(return\)`
		return
	}
	_ = s
}

// A map entry is named by its key; any other index by its container.

func mapKey(m map[string]string) error {
	if m["userID"] == "" { // want `guard on empty userID exits silently`
		return nil
	}
	return nil
}

func sliceIndex(ids []string) error {
	if ids[0] == "" { // want `guard on empty ids exits silently`
		return nil
	}
	return nil
}

// Plurals are reachable.

func plurals(userIDs, IDs, ids, accountIds string) error {
	if userIDs == "" { // want `guard on empty userIDs`
		return nil
	}
	if IDs == "" { // want `guard on empty IDs`
		return nil
	}
	if ids == "" { // want `guard on empty ids`
		return nil
	}
	if accountIds == "" { // want `guard on empty accountIds`
		return nil
	}
	return nil
}

// Not reported: the guard is loud.

func returnsError(userID string) error {
	if userID == "" {
		return errors.New("empty userID")
	}
	return nil
}

func panics(userID string) {
	if userID == "" {
		panic("empty userID")
	}
}

func wraps(userID string) error {
	if userID == "" {
		return fmt.Errorf("empty %s", userID)
	}
	return nil
}

func warnsFirst(userID string) error {
	if userID == "" {
		userID = "default"
		slog.Warn("defaulted", "user_id", userID)
		return nil
	}
	return nil
}

func compoundLoud(ok bool, userID string) error {
	if !ok || userID == "" {
		return errors.New("x")
	}
	return nil
}

func lenLoud(leaseID []byte) error {
	if len(leaseID) == 0 {
		return errors.New("x")
	}
	return nil
}

func switchLoud(userID string) error {
	switch userID {
	case "":
		return errors.New("x")
	}
	return nil
}

func switchWarnsFirst(userID string) error {
	switch userID {
	case "":
		userID = "default"
		slog.Warn("defaulted", "user_id", userID)
		return nil
	}
	return nil
}

// Not reported: the subject is a value, not an identifier. Checking a value
// for emptiness is ordinary input validation.

func plainValue(comment string) error {
	if comment == "" {
		return nil
	}
	return nil
}

func compoundPlainValue(comment string, ok bool) error {
	if !ok || comment == "" {
		return nil
	}
	return nil
}

func plainSlice(rows []int) error {
	if len(rows) == 0 {
		return nil
	}
	return nil
}

func switchPlainValue(comment string) error {
	switch comment {
	case "":
		return nil
	}
	return nil
}

func mapKeyValue(m map[string]string) error {
	if m["comment"] == "" {
		return nil
	}
	return nil
}

func sliceIndexValue(rows []string) error {
	if rows[0] == "" {
		return nil
	}
	return nil
}

// Matching whole words keeps these out: each ends in id or ids as letters.
func notWords(valid, invalid, grid, rapid, solid, grids, solids, rapids string) error {
	if valid == "" || invalid == "" || grid == "" || rapid == "" || solid == "" {
		return nil
	}
	if grids == "" || solids == "" || rapids == "" {
		return nil
	}
	return nil
}

// Not reported: a fork, not a skip.

func hasElse(userID string) error {
	if userID == "" {
		return nil
	} else {
		return nil
	}
}

func compoundElse(ok bool, userID string) error {
	if !ok || userID == "" {
		return nil
	} else {
		return nil
	}
}

// Not reported: not an emptiness test.

func notEmptyTest(userID string) error {
	if userID == "x" {
		return nil
	}
	return nil
}

func lenOne(ids []string) error {
	if len(ids) == 1 {
		return nil
	}
	return nil
}

func lenPositive(ids []string) error {
	if len(ids) > 0 {
		return nil
	}
	return nil
}

func capZero(ids []string) error {
	if cap(ids) == 0 {
		return nil
	}
	return nil
}

// A local named len is not the builtin.
func shadowedLen(ids []string) error {
	len := func([]string) int { return 1 }
	if len(ids) == 0 {
		return nil
	}
	return nil
}

func otherCase(userID string) error {
	switch userID {
	case "x":
		return nil
	}
	return nil
}

func tagless(userID string) error {
	switch {
	case userID == "":
		return nil
	}
	return nil
}

func twoValues(userID string) error {
	switch userID {
	case "", "x":
		return nil
	}
	return nil
}

// Not reported: the body returns a value.
func returnsValue(userID string) (int, error) {
	if userID == "" {
		return 0, nil
	}
	return 1, nil
}

// Not reported: the default suffixes do not include ref.
func notConfigured(tableRef string) error {
	if tableRef == "" {
		return nil
	}
	return nil
}
