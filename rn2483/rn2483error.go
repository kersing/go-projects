// Copyright © 2016 by Jac Kersing
// Use of this source code is governed by the MIT license that can be found in the LICENSE file at the top
// of the github repository for this source

package rn2483

type Rn2483Error struct {
	msg   string
	value int
}

type Error interface {
	error
	RetryAble() bool
	ResetRequired() bool
	RejoinRequired() bool
}

func (e *Rn2483Error) Error() string {
	return e.msg
}

func NewError(str string, val int) error {
	return &Rn2483Error{msg: str, value: val}
}

func (e *Rn2483Error) RetryAble() bool {
	return (e.value == XMIT_RETRY)
}

func (e *Rn2483Error) ResetRequired() bool {
	return (e.value == XMIT_RESET)
}

func (e *Rn2483Error) RejoinRequired() bool {
	return (e.value == XMIT_REJOIN)
}
