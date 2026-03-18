package main

import (
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:    "simple args",
			input:   "arg1 arg2 arg3",
			want:    []string{"arg1", "arg2", "arg3"},
			wantErr: false,
		},
		{
			name:    "double quoted args",
			input:   `"arg with spaces" another-arg`,
			want:    []string{"arg with spaces", "another-arg"},
			wantErr: false,
		},
		{
			name:    "single quoted args",
			input:   `'single quoted' unquoted`,
			want:    []string{"single quoted", "unquoted"},
			wantErr: false,
		},
		{
			name:    "mixed quotes",
			input:   `"double quoted" 'single quoted' plain`,
			want:    []string{"double quoted", "single quoted", "plain"},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   "",
			want:    nil,
			wantErr: false,
		},
		{
			name:    "single arg",
			input:   "only-one",
			want:    []string{"only-one"},
			wantErr: false,
		},
		{
			name:    "multiple spaces",
			input:   "arg1   arg2     arg3",
			want:    []string{"arg1", "arg2", "arg3"},
			wantErr: false,
		},
		{
			name:    "quoted with special chars",
			input:   `"--flag=value" "--another=complex value"`,
			want:    []string{"--flag=value", "--another=complex value"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseArgs(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseKeyValuePairs(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		want    map[string]string
		wantErr bool
	}{
		{
			name:    "simple pairs",
			input:   []string{"KEY1=value1", "KEY2=value2"},
			want:    map[string]string{"KEY1": "value1", "KEY2": "value2"},
			wantErr: false,
		},
		{
			name:    "with spaces",
			input:   []string{"KEY= value with spaces "},
			want:    map[string]string{"KEY": "value with spaces"},
			wantErr: false,
		},
		{
			name:    "double quoted value",
			input:   []string{`KEY="quoted value"`},
			want:    map[string]string{"KEY": "quoted value"},
			wantErr: false,
		},
		{
			name:    "single quoted value",
			input:   []string{"KEY='quoted value'"},
			want:    map[string]string{"KEY": "quoted value"},
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   []string{},
			want:    map[string]string{},
			wantErr: false,
		},
		{
			name:    "value with equals sign",
			input:   []string{"KEY=value=with=equals"},
			want:    map[string]string{"KEY": "value=with=equals"},
			wantErr: false,
		},
		{
			name:    "invalid pair no equals",
			input:   []string{"KEYVALUE"},
			want:    nil,
			wantErr: true,
		},
		{
			name:    "mixed valid and invalid",
			input:   []string{"KEY1=value1", "INVALID", "KEY2=value2"},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseKeyValuePairs(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKeyValuePairs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseKeyValuePairs() = %v, want %v", got, tt.want)
			}
		})
	}
}
