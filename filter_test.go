package main

import (
	"testing"

	"bytes"
	"time"

	"github.com/valyala/fastjson"
)

func Test_filterLog(t *testing.T) {

	testCases := []struct {
		name string
		container_name string
		namespace_name string
		pod_name string
		log string
		expected bool
	}{
		{
			name: "when wildcard is used and log does not match",
			container_name: "container1",
			namespace_name: "namespace1",
			pod_name: "pod1",
			log: "test",
			expected: false,
		},
		{
			name: "when wildcard is used and log matches",
			container_name: "container1",
			namespace_name: "namespace1",
			pod_name: "pod1",
			log: "abc",
			expected: true,
		},
		{
			name: "when no match is found",
			container_name: "a",
			namespace_name: "b",
			pod_name: "c",
			log: "test",
			expected: false,
		},
		{
			name: "when exact match is found",
			container_name: "a",
			namespace_name: "b",
			pod_name: "c",
			log: "def",
			expected: true,
		},
		{
			name: "when exact match is found - negative",
			container_name: "a",
			namespace_name: "b",
			pod_name: "d",
			log: "def",
			expected: false,
		},
		{
			name: "when exact match is found as a substring",
			container_name: "a",
			namespace_name: "b",
			pod_name: "c",
			log: "adefg",
			expected: true,
		},
		{
			name: "when pod name is from a deployment",
			container_name: "a",
			namespace_name: "b",
			pod_name: "document-generation-6499cbb75b-65lmt",
			log: "xyz",
			expected: true,
		},
		{
			name: "when pod name is from a statefulset",
			container_name: "a",
			namespace_name: "b",
			pod_name: "argocd-application-controller-0",
			log: "xyz",
			expected: true,
		},
		{
			name: "when pod name is invalid",
			container_name: "a",
			namespace_name: "b",
			pod_name: "argocd-application-controller-d",
			log: "xyz",
			expected: false,
		},
	}

	var parser fastjson.Parser
	config, err := parser.Parse(`
		{
			"*": {
				"*": {
					"*": "abc",
					"argocd-application-controller": {"pattern": "xyz"},
					"document-generation": {"pattern": "xyz"}
				}
			},
			"a": {
				"b": {
					"c": {"pattern": "def", "invert": false},
					"d": {"pattern": "def", "invert": true}
				}
			}
		}
	`)

	if err != nil {
		t.Errorf("error parsing config: %v", err)
	}

	configSource := ConfigFileConfiguration{
		config: config,
	}


	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			var arena fastjson.Arena
			record:= arena.NewObject()
			record.Set("container_name", arena.NewString(tc.container_name))
			record.Set("namespace_name", arena.NewString(tc.namespace_name))
			record.Set("pod_name", arena.NewString(tc.pod_name))
			record.Set("log", arena.NewString(tc.log))

			actual := filterLog(record, configSource)
			if actual != tc.expected {
				t.Errorf("expected %t, got %t", tc.expected, actual)
			}
		})
	}
}

func Test_extract_pod_name(t *testing.T) {
	
	testCases := []struct {
		name string
		pod_name string
		expected string
	}{
		{
			name: "when pod name is from a deployment",
			pod_name: "document-generation-6499cbb75b-65lmt",
			expected: "document-generation",
		},
		{
			name: "when pod name is from a statefulset",
			pod_name: "argocd-application-controller-0",
			expected: "argocd-application-controller",
		},
		{
			name: "when pod name is invalid",
			pod_name: "argocd-application-controller-d",
			expected: "argocd-application-controller-d",
		},
		{
			name: "when pod name is from a job or daemonset",
			pod_name: "worker-12438-m76v7",
			expected: "worker-12438",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := extractPodName(tc.pod_name)
			if actual != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, actual)
			}
		})
	}
}

func TestGo_filter(t *testing.T){


	testCases := []struct {
		tag string
		record string
		time time.Time
		expected []byte
	}{
		{
			tag: "normal log",
			record: `{"log": "abc"}`,
			time: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: []byte(`{"log":"abc"}`+string(rune(0))),
		},
		{
			tag: "log with special characters",
			record: "{\"log\": \"abc\x07\"}",
			time: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: []byte(`{"log":"abc\\x07"}`+string(rune(0))),
		},
		{
			tag: "log with tab",
			record: "{\"log\": \"abc\t\"}",
			time: time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			expected: []byte(`{"log":"abc\t"}`+string(rune(0))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.tag, func(t *testing.T) {

			actual := go_filter_go(tc.tag, tc.time, tc.record)

			if !bytes.Equal(actual, tc.expected) {
				t.Errorf("expected %s, got %s", tc.expected, actual)
			}
		
		})
	}

}
