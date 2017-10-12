package terraform

import (
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/config/configschema"
)

// ProviderEvalTree returns the evaluation tree for initializing and
// configuring providers.
func ProviderEvalTree(n string, config *config.RawConfig) EvalNode {
	var provider ResourceProvider
	var resourceConfig *ResourceConfig
	var schema *configschema.Block

	seq := make([]EvalNode, 0, 5)
	seq = append(seq, &EvalInitProvider{Name: n})

	// Input stuff
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkInput, walkImport},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n,
					Output: &provider,
				},
				&EvalIf{
					If: func(EvalContext) (bool, error) {
						return config.RequiresSchema(), nil
					},
					Then: &EvalGetProviderSchema{
						ProviderName: n,
						Provider:     &provider,
						Output:       &schema,
					},
				},
				&EvalInterpolate{
					Config: config,
					Schema: &schema,
					Output: &resourceConfig,
				},
				&EvalBuildProviderConfig{
					Provider: n,
					Config:   &resourceConfig,
					Output:   &resourceConfig,
				},
				&EvalInputProvider{
					Name:     n,
					Provider: &provider,
					Config:   &resourceConfig,
				},
			},
		},
	})

	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkValidate},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n,
					Output: &provider,
				},
				&EvalIf{
					If: func(EvalContext) (bool, error) {
						return config.RequiresSchema(), nil
					},
					Then: &EvalGetProviderSchema{
						ProviderName: n,
						Provider:     &provider,
						Output:       &schema,
					},
				},
				&EvalInterpolate{
					Config: config,
					Schema: &schema,
					Output: &resourceConfig,
				},
				&EvalBuildProviderConfig{
					Provider: n,
					Config:   &resourceConfig,
					Output:   &resourceConfig,
				},
				&EvalValidateProvider{
					Provider: &provider,
					Config:   &resourceConfig,
				},
				&EvalSetProviderConfig{
					Provider: n,
					Config:   &resourceConfig,
				},
			},
		},
	})

	// Apply stuff
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkPlan, walkApply, walkDestroy, walkImport},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalGetProvider{
					Name:   n,
					Output: &provider,
				},
				&EvalIf{
					If: func(EvalContext) (bool, error) {
						return config.RequiresSchema(), nil
					},
					Then: &EvalGetProviderSchema{
						ProviderName: n,
						Provider:     &provider,
						Output:       &schema,
					},
				},
				&EvalInterpolate{
					Config: config,
					Schema: &schema,
					Output: &resourceConfig,
				},
				&EvalBuildProviderConfig{
					Provider: n,
					Config:   &resourceConfig,
					Output:   &resourceConfig,
				},
				&EvalSetProviderConfig{
					Provider: n,
					Config:   &resourceConfig,
				},
			},
		},
	})

	// We configure on everything but validate, since validate may
	// not have access to all the variables.
	seq = append(seq, &EvalOpFilter{
		Ops: []walkOperation{walkRefresh, walkPlan, walkApply, walkDestroy, walkImport},
		Node: &EvalSequence{
			Nodes: []EvalNode{
				&EvalConfigProvider{
					Provider: n,
					Config:   &resourceConfig,
				},
			},
		},
	})

	return &EvalSequence{Nodes: seq}
}

// CloseProviderEvalTree returns the evaluation tree for closing
// provider connections that aren't needed anymore.
func CloseProviderEvalTree(n string) EvalNode {
	return &EvalCloseProvider{Name: n}
}
