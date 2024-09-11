// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/AntonBezemskiy/gophermart/internal/repositories (interfaces: AuthInterface,OrdersInterface,BalanceInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"

	repositories "github.com/AntonBezemskiy/gophermart/internal/repositories"
	gomock "github.com/golang/mock/gomock"
)

// MockAuthInterface is a mock of AuthInterface interface.
type MockAuthInterface struct {
	ctrl     *gomock.Controller
	recorder *MockAuthInterfaceMockRecorder
}

// MockAuthInterfaceMockRecorder is the mock recorder for MockAuthInterface.
type MockAuthInterfaceMockRecorder struct {
	mock *MockAuthInterface
}

// NewMockAuthInterface creates a new mock instance.
func NewMockAuthInterface(ctrl *gomock.Controller) *MockAuthInterface {
	mock := &MockAuthInterface{ctrl: ctrl}
	mock.recorder = &MockAuthInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockAuthInterface) EXPECT() *MockAuthInterfaceMockRecorder {
	return m.recorder
}

// Authenticate mocks base method.
func (m *MockAuthInterface) Authenticate(arg0 context.Context, arg1, arg2 string) (bool, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Authenticate", arg0, arg1, arg2)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Authenticate indicates an expected call of Authenticate.
func (mr *MockAuthInterfaceMockRecorder) Authenticate(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Authenticate", reflect.TypeOf((*MockAuthInterface)(nil).Authenticate), arg0, arg1, arg2)
}

// Register mocks base method.
func (m *MockAuthInterface) Register(arg0 context.Context, arg1, arg2 string) (bool, string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Register", arg0, arg1, arg2)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(string)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Register indicates an expected call of Register.
func (mr *MockAuthInterfaceMockRecorder) Register(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Register", reflect.TypeOf((*MockAuthInterface)(nil).Register), arg0, arg1, arg2)
}

// MockOrdersInterface is a mock of OrdersInterface interface.
type MockOrdersInterface struct {
	ctrl     *gomock.Controller
	recorder *MockOrdersInterfaceMockRecorder
}

// MockOrdersInterfaceMockRecorder is the mock recorder for MockOrdersInterface.
type MockOrdersInterfaceMockRecorder struct {
	mock *MockOrdersInterface
}

// NewMockOrdersInterface creates a new mock instance.
func NewMockOrdersInterface(ctrl *gomock.Controller) *MockOrdersInterface {
	mock := &MockOrdersInterface{ctrl: ctrl}
	mock.recorder = &MockOrdersInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockOrdersInterface) EXPECT() *MockOrdersInterfaceMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockOrdersInterface) Get(arg0 context.Context, arg1 string) ([]repositories.Order, int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].([]repositories.Order)
	ret1, _ := ret[1].(int)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Get indicates an expected call of Get.
func (mr *MockOrdersInterfaceMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockOrdersInterface)(nil).Get), arg0, arg1)
}

// Load mocks base method.
func (m *MockOrdersInterface) Load(arg0 context.Context, arg1, arg2 string) (int, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Load", arg0, arg1, arg2)
	ret0, _ := ret[0].(int)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Load indicates an expected call of Load.
func (mr *MockOrdersInterfaceMockRecorder) Load(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Load", reflect.TypeOf((*MockOrdersInterface)(nil).Load), arg0, arg1, arg2)
}

// MockBalanceInterface is a mock of BalanceInterface interface.
type MockBalanceInterface struct {
	ctrl     *gomock.Controller
	recorder *MockBalanceInterfaceMockRecorder
}

// MockBalanceInterfaceMockRecorder is the mock recorder for MockBalanceInterface.
type MockBalanceInterfaceMockRecorder struct {
	mock *MockBalanceInterface
}

// NewMockBalanceInterface creates a new mock instance.
func NewMockBalanceInterface(ctrl *gomock.Controller) *MockBalanceInterface {
	mock := &MockBalanceInterface{ctrl: ctrl}
	mock.recorder = &MockBalanceInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBalanceInterface) EXPECT() *MockBalanceInterfaceMockRecorder {
	return m.recorder
}

// Get mocks base method.
func (m *MockBalanceInterface) Get(arg0 context.Context, arg1 string) (repositories.Balance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", arg0, arg1)
	ret0, _ := ret[0].(repositories.Balance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockBalanceInterfaceMockRecorder) Get(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockBalanceInterface)(nil).Get), arg0, arg1)
}
