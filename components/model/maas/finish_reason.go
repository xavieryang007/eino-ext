package maas

import (
	"fmt"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

var (
	ErrFinishLength        = fmt.Errorf("unexpected message finish reason: %s", model.FinishReasonLength)
	ErrFinishContentFilter = fmt.Errorf("unexpected message finish reason: %s", model.FinishReasonContentFilter)
)

func getUnexpectedFinishReason(reason model.FinishReason) error {
	switch reason {
	case model.FinishReasonLength:
		return ErrFinishLength
	case model.FinishReasonContentFilter:
		return ErrFinishContentFilter
	}

	return nil
}
