package lib

import "errors"

type Error struct {
	Err            error
	DisplayMessage string
}

func (e Error) Error() string {
	return e.Err.Error()
}

func (e Error) DisplayError() error {
	return errors.New(e.DisplayMessage)
}
