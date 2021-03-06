package config

import (
	"fmt"
	"reflect"
	"testing"

	hcl2 "github.com/hashicorp/hcl2/hcl"
	hcl2syntax "github.com/hashicorp/hcl2/hcl/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

func TestConfigValueFromHCL2(t *testing.T) {
	tests := []struct {
		Input cty.Value
		Want  interface{}
	}{
		{
			cty.True,
			true,
		},
		{
			cty.False,
			false,
		},
		{
			cty.NumberIntVal(12),
			int(12),
		},
		{
			cty.NumberFloatVal(12.5),
			float64(12.5),
		},
		{
			cty.StringVal("hello world"),
			"hello world",
		},
		{
			cty.ObjectVal(map[string]cty.Value{
				"name": cty.StringVal("Ermintrude"),
				"age":  cty.NumberIntVal(19),
				"address": cty.ObjectVal(map[string]cty.Value{
					"street": cty.ListVal([]cty.Value{cty.StringVal("421 Shoreham Loop")}),
					"city":   cty.StringVal("Fridgewater"),
					"state":  cty.StringVal("MA"),
					"zip":    cty.StringVal("91037"),
				}),
			}),
			map[string]interface{}{
				"name": "Ermintrude",
				"age":  int(19),
				"address": map[string]interface{}{
					"street": []interface{}{"421 Shoreham Loop"},
					"city":   "Fridgewater",
					"state":  "MA",
					"zip":    "91037",
				},
			},
		},
		{
			cty.MapVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
				"bar": cty.StringVal("baz"),
			}),
			map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
		},
		{
			cty.TupleVal([]cty.Value{
				cty.StringVal("foo"),
				cty.True,
			}),
			[]interface{}{
				"foo",
				true,
			},
		},
		{
			cty.NullVal(cty.String),
			nil,
		},
		{
			cty.UnknownVal(cty.String),
			UnknownVariableValue,
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.Input), func(t *testing.T) {
			got := configValueFromHCL2(test.Input)
			if !reflect.DeepEqual(got, test.Want) {
				t.Errorf("wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v", test.Input, got, test.Want)
			}
		})
	}
}

func TestHCL2ValueFromConfigValue(t *testing.T) {
	tests := []struct {
		Input interface{}
		Want  cty.Value
	}{
		{
			nil,
			cty.NullVal(cty.DynamicPseudoType),
		},
		{
			UnknownVariableValue,
			cty.DynamicVal,
		},
		{
			true,
			cty.True,
		},
		{
			false,
			cty.False,
		},
		{
			int(12),
			cty.NumberIntVal(12),
		},
		{
			int(0),
			cty.Zero,
		},
		{
			float64(12.5),
			cty.NumberFloatVal(12.5),
		},
		{
			"hello world",
			cty.StringVal("hello world"),
		},
		{
			"O\u0308",               // decomposed letter + diacritic
			cty.StringVal("\u00D6"), // NFC-normalized on entry into cty
		},
		{
			[]interface{}{},
			cty.EmptyTupleVal,
		},
		{
			[]interface{}(nil),
			cty.EmptyTupleVal,
		},
		{
			[]interface{}{"hello", "world"},
			cty.TupleVal([]cty.Value{cty.StringVal("hello"), cty.StringVal("world")}),
		},
		{
			map[string]interface{}{},
			cty.EmptyObjectVal,
		},
		{
			map[string]interface{}(nil),
			cty.EmptyObjectVal,
		},
		{
			map[string]interface{}{
				"foo": "bar",
				"bar": "baz",
			},
			cty.ObjectVal(map[string]cty.Value{
				"foo": cty.StringVal("bar"),
				"bar": cty.StringVal("baz"),
			}),
		},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("%#v", test.Input), func(t *testing.T) {
			got := hcl2ValueFromConfigValue(test.Input)
			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\ninput: %#v\ngot:   %#v\nwant:  %#v", test.Input, got, test.Want)
			}
		})
	}
}

func TestHCL2InterpolationFuncs(t *testing.T) {
	// This is not a comprehensive test of all the functions (they are tested
	// in interpolation_funcs_test.go already) but rather just calling a
	// representative set via the HCL2 API to verify that the HCL2-to-HIL
	// function shim is working as expected.
	tests := []struct {
		Expr string
		Want cty.Value
		Err  bool
	}{
		{
			`upper("hello")`,
			cty.StringVal("HELLO"),
			false,
		},
		{
			`abs(-2)`,
			cty.NumberIntVal(2),
			false,
		},
		{
			`abs(-2.5)`,
			cty.NumberFloatVal(2.5),
			false,
		},
		{
			`cidrsubnet("")`,
			cty.DynamicVal,
			true, // not enough arguments
		},
		{
			`cidrsubnet("10.1.0.0/16", 8, 2)`,
			cty.StringVal("10.1.2.0/24"),
			false,
		},
		{
			`concat([])`,
			// Since HIL doesn't maintain element type information for list
			// types, HCL2 can't either without elements to sniff.
			cty.ListValEmpty(cty.DynamicPseudoType),
			false,
		},
		{
			`concat([], [])`,
			cty.ListValEmpty(cty.DynamicPseudoType),
			false,
		},
		{
			`concat(["a"], ["b", "c"])`,
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			false,
		},
		{
			`list()`,
			cty.ListValEmpty(cty.DynamicPseudoType),
			false,
		},
		{
			`list("a", "b", "c")`,
			cty.ListVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			false,
		},
		{
			`list(list("a"), list("b"), list("c"))`,
			// The types emerge here in a bit of a strange tangle because of
			// the guesswork we do when trying to recover lost information from
			// HIL, but the rest of the language doesn't really care whether
			// we use lists or tuples here as long as we are consistent with
			// the type system invariants.
			cty.ListVal([]cty.Value{
				cty.TupleVal([]cty.Value{cty.StringVal("a")}),
				cty.TupleVal([]cty.Value{cty.StringVal("b")}),
				cty.TupleVal([]cty.Value{cty.StringVal("c")}),
			}),
			false,
		},
		{
			`list(list("a"), "b")`,
			cty.DynamicVal,
			true, // inconsistent types
		},
		{
			`length([])`,
			cty.NumberIntVal(0),
			false,
		},
		{
			`length([2])`,
			cty.NumberIntVal(1),
			false,
		},
		{
			`jsonencode(2)`,
			cty.StringVal(`2`),
			false,
		},
		{
			`jsonencode(true)`,
			cty.StringVal(`true`),
			false,
		},
		{
			`jsonencode("foo")`,
			cty.StringVal(`"foo"`),
			false,
		},
		{
			`jsonencode({})`,
			cty.StringVal(`{}`),
			false,
		},
		{
			`jsonencode([1])`,
			cty.StringVal(`[1]`),
			false,
		},
		{
			`jsondecode("{}")`,
			cty.EmptyObjectVal,
			false,
		},
		{
			`jsondecode("[5, true]")[0]`,
			cty.NumberIntVal(5),
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.Expr, func(t *testing.T) {
			expr, diags := hcl2syntax.ParseExpression([]byte(test.Expr), "", hcl2.Pos{Line: 1, Column: 1})
			if len(diags) != 0 {
				for _, diag := range diags {
					t.Logf("- %s", diag)
				}
				t.Fatalf("unexpected diagnostics while parsing expression")
			}

			got, diags := expr.Value(&hcl2.EvalContext{
				Functions: hcl2InterpolationFuncs(),
			})
			gotErr := diags.HasErrors()
			if gotErr != test.Err {
				if test.Err {
					t.Errorf("expected errors but got none")
				} else {
					t.Errorf("unexpected errors")
					for _, diag := range diags {
						t.Logf("- %s", diag)
					}
				}
			}
			if !got.RawEquals(test.Want) {
				t.Errorf("wrong result\nexpr: %s\ngot:  %#v\nwant: %#v", test.Expr, got, test.Want)
			}
		})
	}
}
