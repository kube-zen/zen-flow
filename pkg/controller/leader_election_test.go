/*
Copyright 2025 Kube-ZEN Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"os"
	"testing"

	"k8s.io/client-go/kubernetes/fake"
)

func TestNewLeaderElection(t *testing.T) {
	client := fake.NewSimpleClientset()
	leaderElection, err := NewLeaderElection(client, "test-namespace", "test-name")

	if err != nil {
		t.Fatalf("NewLeaderElection returned error: %v", err)
	}
	if leaderElection == nil {
		t.Fatal("NewLeaderElection returned nil")
	}
	if leaderElection.client != client {
		t.Error("LeaderElection did not set client correctly")
	}
	if leaderElection.namespace != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", leaderElection.namespace)
	}
	if leaderElection.name != "test-name" {
		t.Errorf("Expected name 'test-name', got '%s'", leaderElection.name)
	}
	if leaderElection.identity == "" {
		t.Error("LeaderElection identity should not be empty")
	}
}

func TestNewLeaderElection_UsesPodName(t *testing.T) {
	originalPodName := os.Getenv("POD_NAME")
	defer func() {
		if originalPodName != "" {
			os.Setenv("POD_NAME", originalPodName)
		} else {
			os.Unsetenv("POD_NAME")
		}
	}()

	os.Setenv("POD_NAME", "test-pod")
	client := fake.NewSimpleClientset()
	leaderElection, err := NewLeaderElection(client, "test-namespace", "test-name")

	if err != nil {
		t.Fatalf("NewLeaderElection returned error: %v", err)
	}
	if leaderElection.identity == "" {
		t.Error("LeaderElection identity should not be empty")
	}
}

func TestLeaderElection_IsLeader(t *testing.T) {
	client := fake.NewSimpleClientset()
	leaderElection, err := NewLeaderElection(client, "test-namespace", "test-name")
	if err != nil {
		t.Fatalf("NewLeaderElection returned error: %v", err)
	}

	// Initially should not be leader
	if leaderElection.IsLeader() {
		t.Error("Expected IsLeader to return false initially")
	}
}

func TestLeaderElection_SetCallbacks(t *testing.T) {
	client := fake.NewSimpleClientset()
	leaderElection, err := NewLeaderElection(client, "test-namespace", "test-name")
	if err != nil {
		t.Fatalf("NewLeaderElection returned error: %v", err)
	}

	leaderElection.SetCallbacks(
		func(ctx context.Context) {
			// Callback set
		},
		func() {
			// Callback set
		},
	)

	if leaderElection.onStarted == nil {
		t.Error("onStarted callback should be set")
	}
	if leaderElection.onStopped == nil {
		t.Error("onStopped callback should be set")
	}
}

func TestLeaderElection_Identity(t *testing.T) {
	client := fake.NewSimpleClientset()
	leaderElection, err := NewLeaderElection(client, "test-namespace", "test-name")
	if err != nil {
		t.Fatalf("NewLeaderElection returned error: %v", err)
	}

	identity := leaderElection.Identity()
	if identity == "" {
		t.Error("Identity should not be empty")
	}
}
