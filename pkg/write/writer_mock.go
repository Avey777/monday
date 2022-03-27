// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/write/writer.go

// Package write is a generated GoMock package.
package write

import (
	reflect "reflect"

	config "github.com/eko/monday/pkg/config"
	gomock "github.com/golang/mock/gomock"
)

// MockWriter is a mock of Writer interface.
type MockWriter struct {
	ctrl     *gomock.Controller
	recorder *MockWriterMockRecorder
}

// MockWriterMockRecorder is the mock recorder for MockWriter.
type MockWriterMockRecorder struct {
	mock *MockWriter
}

// NewMockWriter creates a new mock instance.
func NewMockWriter(ctrl *gomock.Controller) *MockWriter {
	mock := &MockWriter{ctrl: ctrl}
	mock.recorder = &MockWriterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWriter) EXPECT() *MockWriterMockRecorder {
	return m.recorder
}

// Write mocks base method.
func (m *MockWriter) Write(application *config.Application) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Write", application)
}

// Write indicates an expected call of Write.
func (mr *MockWriterMockRecorder) Write(application interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Write", reflect.TypeOf((*MockWriter)(nil).Write), application)
}

// WriteAll mocks base method.
func (m *MockWriter) WriteAll() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "WriteAll")
}

// WriteAll indicates an expected call of WriteAll.
func (mr *MockWriterMockRecorder) WriteAll() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "WriteAll", reflect.TypeOf((*MockWriter)(nil).WriteAll))
}
