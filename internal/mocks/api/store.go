// Code generated by MockGen. DO NOT EDIT.
// Source: ./internal/api/store.go

// Package mock_api is a generated GoMock package.
package mock_api

import (
	gomock "github.com/golang/mock/gomock"
	model "github.com/mattermost/awat/model"
	reflect "reflect"
)

// MockStore is a mock of Store interface
type MockStore struct {
	ctrl     *gomock.Controller
	recorder *MockStoreMockRecorder
}

// MockStoreMockRecorder is the mock recorder for MockStore
type MockStoreMockRecorder struct {
	mock *MockStore
}

// NewMockStore creates a new mock instance
func NewMockStore(ctrl *gomock.Controller) *MockStore {
	mock := &MockStore{ctrl: ctrl}
	mock.recorder = &MockStoreMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockStore) EXPECT() *MockStoreMockRecorder {
	return m.recorder
}

// GetTranslation mocks base method
func (m *MockStore) GetTranslation(id string) (*model.Translation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTranslation", id)
	ret0, _ := ret[0].(*model.Translation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTranslation indicates an expected call of GetTranslation
func (mr *MockStoreMockRecorder) GetTranslation(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTranslation", reflect.TypeOf((*MockStore)(nil).GetTranslation), id)
}

// GetTranslationsByInstallation mocks base method
func (m *MockStore) GetTranslationsByInstallation(id string) ([]*model.Translation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetTranslationsByInstallation", id)
	ret0, _ := ret[0].([]*model.Translation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetTranslationsByInstallation indicates an expected call of GetTranslationsByInstallation
func (mr *MockStoreMockRecorder) GetTranslationsByInstallation(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetTranslationsByInstallation", reflect.TypeOf((*MockStore)(nil).GetTranslationsByInstallation), id)
}

// GetAllTranslations mocks base method
func (m *MockStore) GetAllTranslations() ([]*model.Translation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllTranslations")
	ret0, _ := ret[0].([]*model.Translation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllTranslations indicates an expected call of GetAllTranslations
func (mr *MockStoreMockRecorder) GetAllTranslations() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllTranslations", reflect.TypeOf((*MockStore)(nil).GetAllTranslations))
}

// CreateTranslation mocks base method
func (m *MockStore) CreateTranslation(t *model.Translation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTranslation", t)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateTranslation indicates an expected call of CreateTranslation
func (mr *MockStoreMockRecorder) CreateTranslation(t interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTranslation", reflect.TypeOf((*MockStore)(nil).CreateTranslation), t)
}

// UpdateTranslation mocks base method
func (m *MockStore) UpdateTranslation(t *model.Translation) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateTranslation", t)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateTranslation indicates an expected call of UpdateTranslation
func (mr *MockStoreMockRecorder) UpdateTranslation(t interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTranslation", reflect.TypeOf((*MockStore)(nil).UpdateTranslation), t)
}

// GetAndClaimNextReadyImport mocks base method
func (m *MockStore) GetAndClaimNextReadyImport(provisionerID string) (*model.Import, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAndClaimNextReadyImport", provisionerID)
	ret0, _ := ret[0].(*model.Import)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAndClaimNextReadyImport indicates an expected call of GetAndClaimNextReadyImport
func (mr *MockStoreMockRecorder) GetAndClaimNextReadyImport(provisionerID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAndClaimNextReadyImport", reflect.TypeOf((*MockStore)(nil).GetAndClaimNextReadyImport), provisionerID)
}

// GetAllImports mocks base method
func (m *MockStore) GetAllImports() ([]*model.Import, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllImports")
	ret0, _ := ret[0].([]*model.Import)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAllImports indicates an expected call of GetAllImports
func (mr *MockStoreMockRecorder) GetAllImports() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllImports", reflect.TypeOf((*MockStore)(nil).GetAllImports))
}

// GetImport mocks base method
func (m *MockStore) GetImport(id string) (*model.Import, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImport", id)
	ret0, _ := ret[0].(*model.Import)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImport indicates an expected call of GetImport
func (mr *MockStoreMockRecorder) GetImport(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImport", reflect.TypeOf((*MockStore)(nil).GetImport), id)
}

// GetImportsInProgress mocks base method
func (m *MockStore) GetImportsInProgress() ([]*model.Import, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImportsInProgress")
	ret0, _ := ret[0].([]*model.Import)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImportsInProgress indicates an expected call of GetImportsInProgress
func (mr *MockStoreMockRecorder) GetImportsInProgress() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImportsInProgress", reflect.TypeOf((*MockStore)(nil).GetImportsInProgress))
}

// GetImportsByInstallation mocks base method
func (m *MockStore) GetImportsByInstallation(id string) ([]*model.Import, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImportsByInstallation", id)
	ret0, _ := ret[0].([]*model.Import)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImportsByInstallation indicates an expected call of GetImportsByInstallation
func (mr *MockStoreMockRecorder) GetImportsByInstallation(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImportsByInstallation", reflect.TypeOf((*MockStore)(nil).GetImportsByInstallation), id)
}

// GetImportsByTranslation mocks base method
func (m *MockStore) GetImportsByTranslation(id string) ([]*model.Import, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetImportsByTranslation", id)
	ret0, _ := ret[0].([]*model.Import)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetImportsByTranslation indicates an expected call of GetImportsByTranslation
func (mr *MockStoreMockRecorder) GetImportsByTranslation(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetImportsByTranslation", reflect.TypeOf((*MockStore)(nil).GetImportsByTranslation), id)
}

// UpdateImport mocks base method
func (m *MockStore) UpdateImport(imp *model.Import) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateImport", imp)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateImport indicates an expected call of UpdateImport
func (mr *MockStoreMockRecorder) UpdateImport(imp interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateImport", reflect.TypeOf((*MockStore)(nil).UpdateImport), imp)
}

// GetUpload mocks base method
func (m *MockStore) GetUpload(id string) (*model.Upload, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetUpload", id)
	ret0, _ := ret[0].(*model.Upload)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetUpload indicates an expected call of GetUpload
func (mr *MockStoreMockRecorder) GetUpload(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetUpload", reflect.TypeOf((*MockStore)(nil).GetUpload), id)
}

// CreateUpload mocks base method
func (m *MockStore) CreateUpload(id string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateUpload", id)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateUpload indicates an expected call of CreateUpload
func (mr *MockStoreMockRecorder) CreateUpload(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateUpload", reflect.TypeOf((*MockStore)(nil).CreateUpload), id)
}

// CompleteUpload mocks base method
func (m *MockStore) CompleteUpload(uploadID, errorMessage string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CompleteUpload", uploadID, errorMessage)
	ret0, _ := ret[0].(error)
	return ret0
}

// CompleteUpload indicates an expected call of CompleteUpload
func (mr *MockStoreMockRecorder) CompleteUpload(uploadID, errorMessage interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CompleteUpload", reflect.TypeOf((*MockStore)(nil).CompleteUpload), uploadID, errorMessage)
}
