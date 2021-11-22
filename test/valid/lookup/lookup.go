package lookup

import "context"

// Request is the lookup request.
type Request struct {
	ID string
}

// Response is the lookup response.
type Response struct {
	Name string
}

// Func is the lookup function.
type Func func(context.Context, *Request) (*Response, error)

// Service is the lookup service.
type Service interface {
	Lookup(context.Context, *Request) (*Response, error)
}

type service struct{}

// New create a new lookup service.
func New() Service {
	return &service{}
}

func (s *service) Lookup(ctx context.Context, req *Request) (*Response, error) {
	return &Response{
		Name: "Jane",
	}, nil
}
