//go:build !linux

package elevation

import "context"

func RunHelper(context.Context) error            { return ErrUnavailable }
func ledgerStatus(Policy, Grant) (Result, error) { return Result{}, ErrUnavailable }
