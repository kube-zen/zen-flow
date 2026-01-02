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
	"testing"
)

// TestSetupController is skipped because it requires a fully functional manager
// which is complex to mock. The SetupController function is integration-tested
// in the main application.
func TestSetupController(t *testing.T) {
	t.Skip("Skipping - requires full manager implementation which is complex to mock")
}

// TestSetupControllerWithReconciler is skipped because it requires a fully functional manager
// which is complex to mock. The SetupControllerWithReconciler function is integration-tested
// in the main application.
func TestSetupControllerWithReconciler(t *testing.T) {
	t.Skip("Skipping - requires full manager implementation which is complex to mock")
}

