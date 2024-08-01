// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/watch/watcher.go
//
// Generated by this command:
//
//	mockgen -source=pkg/watch/watcher.go -destination=pkg/watch/watcher_mock.go -package=watch
//

// Package watch is a generated GoMock package.
package watch

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockWatcher is a mock of Watcher interface.
type MockWatcher struct {
	ctrl     *gomock.Controller
	recorder *MockWatcherMockRecorder
}

// MockWatcherMockRecorder is the mock recorder for MockWatcher.
type MockWatcherMockRecorder struct {
	mock *MockWatcher
}

// NewMockWatcher creates a new mock instance.
func NewMockWatcher(ctrl *gomock.Controller) *MockWatcher {
	mock := &MockWatcher{ctrl: ctrl}
	mock.recorder = &MockWatcherMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWatcher) EXPECT() *MockWatcherMockRecorder {
	return m.recorder
}

// Stop mocks base method.
func (m *MockWatcher) Stop() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Stop")
	ret0, _ := ret[0].(error)
	return ret0
}

// Stop indicates an expected call of Stop.
func (mr *MockWatcherMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockWatcher)(nil).Stop))
}

// Watch mocks base method.
func (m *MockWatcher) Watch(ctx context.Context) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Watch", ctx)
}

// Watch indicates an expected call of Watch.
func (mr *MockWatcherMockRecorder) Watch(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockWatcher)(nil).Watch), ctx)
}
