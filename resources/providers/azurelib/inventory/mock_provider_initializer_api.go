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

package inventory

import (
	context "context"

	auth "github.com/elastic/cloudbeat/resources/providers/azurelib/auth"

	logp "github.com/elastic/elastic-agent-libs/logp"

	mock "github.com/stretchr/testify/mock"
)

// MockProviderInitializerAPI is an autogenerated mock type for the ProviderInitializerAPI type
type MockProviderInitializerAPI struct {
	mock.Mock
}

type MockProviderInitializerAPI_Expecter struct {
	mock *mock.Mock
}

func (_m *MockProviderInitializerAPI) EXPECT() *MockProviderInitializerAPI_Expecter {
	return &MockProviderInitializerAPI_Expecter{mock: &_m.Mock}
}

// Init provides a mock function with given fields: ctx, log, azureConfig
func (_m *MockProviderInitializerAPI) Init(ctx context.Context, log *logp.Logger, azureConfig auth.AzureFactoryConfig) (ServiceAPI, error) {
	ret := _m.Called(ctx, log, azureConfig)

	var r0 ServiceAPI
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, *logp.Logger, auth.AzureFactoryConfig) (ServiceAPI, error)); ok {
		return rf(ctx, log, azureConfig)
	}
	if rf, ok := ret.Get(0).(func(context.Context, *logp.Logger, auth.AzureFactoryConfig) ServiceAPI); ok {
		r0 = rf(ctx, log, azureConfig)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(ServiceAPI)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, *logp.Logger, auth.AzureFactoryConfig) error); ok {
		r1 = rf(ctx, log, azureConfig)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// MockProviderInitializerAPI_Init_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Init'
type MockProviderInitializerAPI_Init_Call struct {
	*mock.Call
}

// Init is a helper method to define mock.On call
//   - ctx context.Context
//   - log *logp.Logger
//   - azureConfig auth.AzureFactoryConfig
func (_e *MockProviderInitializerAPI_Expecter) Init(ctx interface{}, log interface{}, azureConfig interface{}) *MockProviderInitializerAPI_Init_Call {
	return &MockProviderInitializerAPI_Init_Call{Call: _e.mock.On("Init", ctx, log, azureConfig)}
}

func (_c *MockProviderInitializerAPI_Init_Call) Run(run func(ctx context.Context, log *logp.Logger, azureConfig auth.AzureFactoryConfig)) *MockProviderInitializerAPI_Init_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(context.Context), args[1].(*logp.Logger), args[2].(auth.AzureFactoryConfig))
	})
	return _c
}

func (_c *MockProviderInitializerAPI_Init_Call) Return(_a0 ServiceAPI, _a1 error) *MockProviderInitializerAPI_Init_Call {
	_c.Call.Return(_a0, _a1)
	return _c
}

func (_c *MockProviderInitializerAPI_Init_Call) RunAndReturn(run func(context.Context, *logp.Logger, auth.AzureFactoryConfig) (ServiceAPI, error)) *MockProviderInitializerAPI_Init_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockProviderInitializerAPI creates a new instance of MockProviderInitializerAPI. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockProviderInitializerAPI(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockProviderInitializerAPI {
	mock := &MockProviderInitializerAPI{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
