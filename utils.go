package main

import "fmt"

func NewEhallSystemError(err string, eid int) error {
	return fmt.Errorf("%w%s(%d)", EhallSystemError, err, eid)
}
