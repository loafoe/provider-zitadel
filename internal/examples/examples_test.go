/*
Copyright 2025 The Crossplane Authors.

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

// Package examples checks that every shipped example is a manifest that could
// actually be applied.
//
// The packaging step parses every file in examples/ to put it in the provider
// bundle, so an example that does not parse fails the build rather than
// anything to do with the code in it. That is a long way round to find a typo,
// and it was found that way: a multi-line XML block in an identity provider
// example lost its indentation, which ended the YAML scalar and left the rest of
// the document to be read as a key. CI reported
//
//	failed to parse examples: error converting YAML to JSON: yaml: line 22:
//	could not find expected ':'
//
// and nothing else in the pipeline had an opinion.
//
// The same library the packaging step uses is used here, so this is a check by
// the thing that failed rather than by a stand-in that might disagree with it.
package examples

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sigs.k8s.io/yaml"
)

// examplesDir is where the manifests are, relative to this package.
const examplesDir = "../../examples"

func TestEveryExampleParses(t *testing.T) {
	files, err := filepath.Glob(filepath.Join(examplesDir, "*.yaml"))
	if err != nil {
		t.Fatal(err)
	}

	if len(files) == 0 {
		t.Fatal("no examples found; the packaging step parses every one of them, " +
			"so finding none here means the path is wrong rather than that there are none")
	}

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatal(err)
			}

			if strings.TrimSpace(string(b)) == "" {
				t.Fatal("the example is empty")
			}

			var out map[string]any
			if err := yaml.Unmarshal(b, &out); err != nil {
				t.Fatalf("the packaging build parses this file and will fail: %v", err)
			}

			if out["apiVersion"] == nil || out["kind"] == nil {
				t.Errorf("no apiVersion or kind: the example names no resource")
			}
		})
	}
}
