package model

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGoods_Scan(t *testing.T) {
	tests := []struct {
		name        string
		input       interface{}
		expected    Goods
		expectError bool
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: Goods{},
		},
		{
			name:     "empty bytes",
			input:    []byte{},
			expected: Goods{},
		},
		{
			name:     "null string",
			input:    "null",
			expected: Goods{},
		},
		{
			name:  "valid JSON array",
			input: []byte(`[{"description":"Good 1","price":1000},{"description":"Good 2","price":25}]`),
			expected: Goods{
				{Description: "Good 1", Price: 1000},
				{Description: "Good 2", Price: 25},
			},
		},
		{
			name:        "invalid JSON",
			input:       []byte(`invalid json`),
			expectError: true,
		},
		{
			name:        "unsupported type",
			input:       123,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var goods Goods
			err := goods.Scan(tt.input)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, goods)
			}
		})
	}
}

func TestOrder_JSONMarshal(t *testing.T) {
	tests := []struct {
		name     string
		order    Order
		expected string
	}{
		{
			name: "order with accrual",
			order: Order{
				Number:  "12345",
				Status:  StatusProcessed,
				Accrual: 100,
				Goods: Goods{
					{Description: "Good 1", Price: 1000},
				},
			},
			expected: `{"order":"12345","status":"PROCESSED","accrual":100}`,
		},
		{
			name: "order without accrual",
			order: Order{
				Number: "67890",
				Status: StatusRegistered,
				Goods:  Goods{},
			},
			expected: `{"order":"67890","status":"REGISTERED"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.order)
			assert.NoError(t, err)

			var expectedObj, actualObj interface{}
			err = json.Unmarshal([]byte(tt.expected), &expectedObj)
			assert.NoError(t, err)
			err = json.Unmarshal(data, &actualObj)
			assert.NoError(t, err)

			assert.Equal(t, expectedObj, actualObj)
		})
	}
}
