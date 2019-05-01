// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/rueian/godemand/types (interfaces: Controller)

// Package types is a generated GoMock package.
package types

import (
	gomock "github.com/golang/mock/gomock"
	types "github.com/rueian/godemand/types"
	reflect "reflect"
)

// MockController is a mock of Controller interface
type MockController struct {
	ctrl     *gomock.Controller
	recorder *MockControllerMockRecorder
}

// MockControllerMockRecorder is the mock recorder for MockController
type MockControllerMockRecorder struct {
	mock *MockController
}

// NewMockController creates a new mock instance
func NewMockController(ctrl *gomock.Controller) *MockController {
	mock := &MockController{ctrl: ctrl}
	mock.recorder = &MockControllerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockController) EXPECT() *MockControllerMockRecorder {
	return m.recorder
}

// FindResource mocks base method
func (m *MockController) FindResource(arg0 types.ResourcePool, arg1 map[string]interface{}) (types.Resource, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FindResource", arg0, arg1)
	ret0, _ := ret[0].(types.Resource)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FindResource indicates an expected call of FindResource
func (mr *MockControllerMockRecorder) FindResource(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FindResource", reflect.TypeOf((*MockController)(nil).FindResource), arg0, arg1)
}

// SyncResource mocks base method
func (m *MockController) SyncResource(arg0 types.Resource, arg1 map[string]interface{}) (types.Resource, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SyncResource", arg0, arg1)
	ret0, _ := ret[0].(types.Resource)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SyncResource indicates an expected call of SyncResource
func (mr *MockControllerMockRecorder) SyncResource(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SyncResource", reflect.TypeOf((*MockController)(nil).SyncResource), arg0, arg1)
}