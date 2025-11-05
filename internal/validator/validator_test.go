package validator_test

import (
	"github.com/alex-storchak/go-musthave-group-diploma/internal/validator"
	"testing"

	"github.com/alex-storchak/go-musthave-group-diploma/internal/validator/mocks"
	"github.com/stretchr/testify/assert"
)

func TestIsValid(t *testing.T) {
	tests := []struct {
		name             string
		setupMock        func(*mocks.MockValidator)
		expectedProblems validator.Problems
		expectError      bool
		expectedErr      error
	}{
		{
			name: "valid object - no problems",
			setupMock: func(m *mocks.MockValidator) {
				m.EXPECT().Valid().Return(validator.Problems{})
			},
			expectedProblems: validator.Problems{},
			expectedErr:      nil,
			expectError:      false,
		},
		{
			name: "invalid object - with problems",
			setupMock: func(m *mocks.MockValidator) {
				m.EXPECT().Valid().Return(validator.Problems{
					"field1": "error 1",
					"field2": "error 2",
				})
			},
			expectedProblems: validator.Problems{
				"field1": "error 1",
				"field2": "error 2",
			},
			expectError: true,
			expectedErr: validator.ErrTypeInvalid,
		},
		{
			name: "invalid object - single problem",
			setupMock: func(m *mocks.MockValidator) {
				m.On("Valid").Return(validator.Problems{"email": "invalid email format"})
			},
			expectedProblems: validator.Problems{"email": "invalid email format"},
			expectError:      true,
			expectedErr:      validator.ErrTypeInvalid,
		},
		{
			name: "empty problems map",
			setupMock: func(m *mocks.MockValidator) {
				m.On("Valid").Return(validator.Problems{})
			},
			expectedProblems: validator.Problems{},
			expectError:      false,
			expectedErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockValidator := mocks.NewMockValidator(t)
			tt.setupMock(mockValidator)

			problems, err := validator.IsValid(mockValidator)

			if tt.expectError {
				assert.ErrorIs(t, err, tt.expectedErr)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedProblems, problems)
		})
	}
}
