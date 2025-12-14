package action

import "context"

type Action interface {
	Name() string

	Parameters() map[string]interface{}

	Validate() error

	Run(ctx context.Context) error
}
