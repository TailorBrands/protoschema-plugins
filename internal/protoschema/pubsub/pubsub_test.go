// Copyright 2024-2025 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pubsub

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/bufbuild/protoschema-plugins/internal/protoschema/golden"
	"github.com/bufbuild/protoschema-plugins/internal/protoschema/normalize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestPubSubGolden(t *testing.T) {
	t.Parallel()
	dirPath := filepath.FromSlash("../../testdata/pubsub")
	testDescs, err := golden.GetTestDescriptors("../../testdata")
	require.NoError(t, err)
	for _, testDesc := range testDescs {
		filePath := filepath.Join(dirPath, string(testDesc.FullName()))
		data, err := Generate(testDesc)
		require.NoError(t, err)
		err = golden.CheckGolden(fmt.Sprintf("%s.%s", filePath, FileExtension), data)
		require.NoError(t, err)
	}
}

func TestPubSubPreserveOpenEnums(t *testing.T) {
	t.Parallel()
	testDescs, err := golden.GetTestDescriptors("../../testdata")
	require.NoError(t, err)

	// Find TestAllTypes which has open enums (proto3 syntax).
	testAllTypesDesc := findDescByName(testDescs, "bufext.cel.expr.conformance.proto3.TestAllTypes")
	require.NotNil(t, testAllTypesDesc, "TestAllTypes descriptor not found")

	t.Run("default_converts_open_enums_to_int32", func(t *testing.T) {
		t.Parallel()
		data, err := Generate(testAllTypesDesc)
		require.NoError(t, err)
		// Default behavior: open enum fields are converted to int32.
		assert.Contains(t, data, "optional int32 standalone_enum")
		assert.Contains(t, data, "int32 single_nested_enum")
		assert.Contains(t, data, "repeated int32 repeated_nested_enum")
	})

	t.Run("preserve_open_enums_keeps_enum_type", func(t *testing.T) {
		t.Parallel()
		data, err := Generate(testAllTypesDesc, normalize.WithPreserveOpenEnums())
		require.NoError(t, err)
		// With preserve open enums: enum field types are preserved, not converted to int32.
		assert.NotContains(t, data, "optional int32 standalone_enum")
		assert.NotContains(t, data, "int32 single_nested_enum")
		assert.NotContains(t, data, "repeated int32 repeated_nested_enum")
		// The fields should reference the enum type instead.
		assert.Contains(t, data, "NestedEnum standalone_enum")
		assert.Contains(t, data, "NestedEnum single_nested_enum")
		assert.Contains(t, data, "repeated NestedEnum repeated_nested_enum")
	})

	// Verify the option also works for a message with no open enums (no change expected).
	productDesc := findDescByName(testDescs, "buf.protoschema.test.v1.Product")
	require.NotNil(t, productDesc, "Product descriptor not found")

	t.Run("preserve_open_enums_no_change_for_messages_without_enums", func(t *testing.T) {
		t.Parallel()
		defaultData, err := Generate(productDesc)
		require.NoError(t, err)
		preservedData, err := Generate(productDesc, normalize.WithPreserveOpenEnums())
		require.NoError(t, err)
		assert.Equal(t, defaultData, preservedData)
	})
}

func findDescByName(descs []protoreflect.MessageDescriptor, name protoreflect.FullName) protoreflect.MessageDescriptor {
	for _, desc := range descs {
		if desc.FullName() == name {
			return desc
		}
	}
	return nil
}
