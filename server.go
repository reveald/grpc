package grpc

import (
	context "context"
	"fmt"
	"net"

	"github.com/reveald/reveald"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	UnimplementedRevealdServiceServer
	backend   reveald.Backend
	endpoints map[string]*reveald.Endpoint
	conv      func(map[string]interface{}) (proto.Message, bool)
}

type ServerOption func(*Server) error

func WithConverter(fn func(map[string]interface{}) (proto.Message, bool)) ServerOption {
	return func(s *Server) error {
		s.conv = fn
		return nil
	}
}

func WithEndpoint(name string, indices reveald.Indices, features ...reveald.Feature) ServerOption {
	return func(s *Server) error {
		ep := reveald.NewEndpoint(s.backend, indices)
		err := ep.Register(features...)
		if err != nil {
			return err
		}

		s.Endpoint(name, ep)
		return nil
	}
}

func New(backend reveald.Backend, opts ...ServerOption) (*Server, error) {
	s := &Server{}
	s.backend = backend
	s.endpoints = make(map[string]*reveald.Endpoint)
	s.conv = func(_ map[string]interface{}) (proto.Message, bool) {
		return nil, false
	}

	for _, opt := range opts {
		err := opt(s)
		if err != nil {
			return nil, err
		}
	}

	return s, nil
}

func (s *Server) Endpoint(name string, ep *reveald.Endpoint) {
	s.endpoints[name] = ep
}

func (s *Server) NewRequest(ctx context.Context, in *Request) (*Result, error) {
	rqt := mapRequest(in)

	var endpoint *reveald.Endpoint = nil
	for n, ep := range s.endpoints {
		if n == in.Target {
			endpoint = ep
			break
		}
	}

	if endpoint == nil {
		return nil, fmt.Errorf("no endpoint found by name %s", in.Target)
	}

	rsp, err := endpoint.Execute(ctx, rqt)
	if err != nil {
		return nil, err
	}

	out := mapResponse(rsp, s.conv)
	return out, nil
}

func (s *Server) ListenAndServe(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	srv := grpc.NewServer()
	RegisterRevealdServiceServer(srv, s)
	return srv.Serve(l)
}
