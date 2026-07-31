package analysis_test

import (
	"go/types"
	"testing"

	"github.com/unbound-force/gaze/internal/analysis"
)

func TestMatchesWriteSignature(t *testing.T) {
	byteSlice := types.NewSlice(types.Typ[types.Byte])
	errorType := types.Universe.Lookup("error").Type()

	mkSig := func(params []*types.Var, results []*types.Var) *types.Signature {
		return types.NewSignatureType(
			nil, nil, nil,
			types.NewTuple(params...),
			types.NewTuple(results...),
			false,
		)
	}

	tests := []struct {
		name string
		sig  *types.Signature
		want bool
	}{
		{
			name: "valid Write([]byte)(int,error)",
			sig: mkSig(
				[]*types.Var{types.NewVar(0, nil, "p", byteSlice)},
				[]*types.Var{
					types.NewVar(0, nil, "", types.Typ[types.Int]),
					types.NewVar(0, nil, "", errorType),
				},
			),
			want: true,
		},
		{
			name: "wrong param: Write(string)(int,error)",
			sig: mkSig(
				[]*types.Var{types.NewVar(0, nil, "p", types.Typ[types.String])},
				[]*types.Var{
					types.NewVar(0, nil, "", types.Typ[types.Int]),
					types.NewVar(0, nil, "", errorType),
				},
			),
			want: false,
		},
		{
			name: "wrong return int type: Write([]byte)(int64,error)",
			sig: mkSig(
				[]*types.Var{types.NewVar(0, nil, "p", byteSlice)},
				[]*types.Var{
					types.NewVar(0, nil, "", types.Typ[types.Int64]),
					types.NewVar(0, nil, "", errorType),
				},
			),
			want: false,
		},
		{
			name: "extra param: Write([]byte,int)(int,error)",
			sig: mkSig(
				[]*types.Var{
					types.NewVar(0, nil, "p", byteSlice),
					types.NewVar(0, nil, "n", types.Typ[types.Int]),
				},
				[]*types.Var{
					types.NewVar(0, nil, "", types.Typ[types.Int]),
					types.NewVar(0, nil, "", errorType),
				},
			),
			want: false,
		},
		{
			name: "missing error return: Write([]byte) int",
			sig: mkSig(
				[]*types.Var{types.NewVar(0, nil, "p", byteSlice)},
				[]*types.Var{
					types.NewVar(0, nil, "", types.Typ[types.Int]),
				},
			),
			want: false,
		},
		{
			name: "no params: Write()(int,error)",
			sig: mkSig(
				nil,
				[]*types.Var{
					types.NewVar(0, nil, "", types.Typ[types.Int]),
					types.NewVar(0, nil, "", errorType),
				},
			),
			want: false,
		},
		{
			name: "wrong error return: Write([]byte)(int,string)",
			sig: mkSig(
				[]*types.Var{types.NewVar(0, nil, "p", byteSlice)},
				[]*types.Var{
					types.NewVar(0, nil, "", types.Typ[types.Int]),
					types.NewVar(0, nil, "", types.Typ[types.String]),
				},
			),
			want: false,
		},
		{
			name: "no results: Write([]byte)",
			sig: mkSig(
				[]*types.Var{types.NewVar(0, nil, "p", byteSlice)},
				nil,
			),
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := analysis.MatchesWriteSignature(tc.sig)
			if got != tc.want {
				t.Errorf("MatchesWriteSignature() = %v, want %v", got, tc.want)
			}
		})
	}
}
