// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Code generated by mockery v2.33.3. DO NOT EDIT.

package fetching

import (
	context "context"

	mock "github.com/stretchr/testify/mock"
)

// MockFetcher is an autogenerated mock type for the Fetcher type
type MockFetcher struct {
	mock.Mock
}

type MockFetcher_Expecter struct {
	mock *mock.Mock
}

func (_m *MockFetcher) EXPECT() *MockFetcher_Expecter {
	return &MockFetcher_Expecter{mock: &_m.Mock}
}

// Fetch provides a mock function with given fields: _a0, _a1
func (_m *MockFetcher) Fetch(_a0 context.Context, _a1 CycleMetadata) error {
	ret := _m.Called(_a0, _a1)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, CycleMetadata) error); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockFetcher_Fetch_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Fetch'
type MockFetcher_Fetch_Call struct {
	*mock.Call
}

// Fetch is a helper method to define mock.On call
//   - _a0 context.Context
//   - _a1 CycleMetadata
func (_e *MockFetcher_Expecter) Fetch(_a0 interface{}, _a1 interface{}) *MockFetcher_Fetch_Call {
	return &MockFetcher_Fetch_Call{Call: _e.mock.On("Fetch", _a0, _a1)}
}

func (_c *MockFetcher_Fetch_Call) Run(run func(_a0 context.Context, _a1 CycleMetadata)) *MockFetcher_Fetch_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(CycleMetadata))
	})
	return _c
}

func (_c *MockFetcher_Fetch_Call) Return(_a0 error) *MockFetcher_Fetch_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockFetcher_Fetch_Call) RunAndReturn(run func(context.Context, CycleMetadata) error) *MockFetcher_Fetch_Call {
	_c.Call.Return(run)
	return _c
}

// Stop provides a mock function with given fields:
func (_m *MockFetcher) Stop() {
	_m.Called()
}

// MockFetcher_Stop_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Stop'
type MockFetcher_Stop_Call struct {
	*mock.Call
}

// Stop is a helper method to define mock.On call
func (_e *MockFetcher_Expecter) Stop() *MockFetcher_Stop_Call {
	return &MockFetcher_Stop_Call{Call: _e.mock.On("Stop")}
}

func (_c *MockFetcher_Stop_Call) Run(run func()) *MockFetcher_Stop_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *MockFetcher_Stop_Call) Return() *MockFetcher_Stop_Call {
	_c.Call.Return()
	return _c
}

func (_c *MockFetcher_Stop_Call) RunAndReturn(run func()) *MockFetcher_Stop_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockFetcher creates a new instance of MockFetcher. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockFetcher(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockFetcher {
	mock := &MockFetcher{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
