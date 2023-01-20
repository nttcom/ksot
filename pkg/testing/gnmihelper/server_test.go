/*
 Copyright (c) 2022-2023 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package gnmihelper_test

import (
	"context"
	"testing"
	"time"

	"github.com/nttcom/kuesta/pkg/testing/gnmihelper"
	gclient "github.com/openconfig/gnmi/client"
	gnmiclient "github.com/openconfig/gnmi/client/gnmi"
	pb "github.com/openconfig/gnmi/proto/gnmi"
	"github.com/stretchr/testify/assert"
)

func TestNewServer(t *testing.T) {
	getCalled := false
	setCalled := false
	capabilitiesCalled := false
	subscribeCalled := false
	m := &gnmihelper.GnmiMock{
		GetHandler: func(ctx context.Context, request *pb.GetRequest) (*pb.GetResponse, error) {
			getCalled = true
			return &pb.GetResponse{}, nil
		},
		SetHandler: func(ctx context.Context, request *pb.SetRequest) (*pb.SetResponse, error) {
			setCalled = true
			return &pb.SetResponse{}, nil
		},
		CapabilitiesHandler: func(ctx context.Context, request *pb.CapabilityRequest) (*pb.CapabilityResponse, error) {
			capabilitiesCalled = true
			return &pb.CapabilityResponse{}, nil
		},
		SubscribeHandler: func(stream pb.GNMI_SubscribeServer) error {
			subscribeCalled = true
			_ = stream.Send(&pb.SubscribeResponse{})
			return nil
		},
	}
	ctx := context.Background()
	gs, conn := gnmihelper.NewGnmiServer(ctx, m)
	defer gs.Stop()

	client, err := gnmiclient.NewFromConn(ctx, conn, gclient.Destination{})
	if err != nil {
		t.Fatal(err)
	}

	client.Get(ctx, &pb.GetRequest{})
	client.Set(ctx, &pb.SetRequest{})
	client.Capabilities(ctx, &pb.CapabilityRequest{})
	q := gclient.Query{
		Type:                gclient.Stream,
		NotificationHandler: nil,
	}
	client.Subscribe(ctx, q)

	assert.True(t, getCalled)
	assert.True(t, setCalled)
	assert.True(t, capabilitiesCalled)

	isSubscribeCalled := func() bool {
		return subscribeCalled
	}

	assert.Eventually(t, isSubscribeCalled, time.Millisecond*100, time.Millisecond*10)
}
