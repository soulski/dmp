package util

import (
	"fmt"
)

type InvalidArgument struct {
	Name  string
	Value string
}

func CreateInvalidArgs(name string, value string) error {
	return &InvalidArgument{Name: name, Value: value}
}

func (e *InvalidArgument) Error() string {
	return fmt.Sprintf("Invalid argument '%s' with value '%s'", e.Name, e.Value)
}

type MessageToLongErr struct {
	actual int64
	expect int64
}

func CreateMsgTooLongErr(expect int64, actual int64) error {
	return &MessageToLongErr{expect: expect, actual: actual}
}

func (e *MessageToLongErr) Error() string {
	return fmt.Sprintf("Message to long expect %d but got %d", e.expect, e.actual)
}

type InvalidProtocol struct {
	cause string
}

func CreateInvalidProtocol(cause string) error {
	return &InvalidProtocol{cause: cause}
}

func (e *InvalidProtocol) Error() string {
	return fmt.Sprintf("Invalid protocol : %s \n", e.cause)
}

type IncompleteMultiErr struct {
	addrs []string
}

func CreateIncompleteMultiErr(failnodes []string) *IncompleteMultiErr {
	return &IncompleteMultiErr{
		addrs: failnodes,
	}
}

func (e *IncompleteMultiErr) Error() string {
	return fmt.Sprintf("fail sending to this nodes %s \n", e.addrs)
}
