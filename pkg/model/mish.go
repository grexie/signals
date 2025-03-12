package model

import (
	"fmt"

	"gorgonia.org/gorgonia"
)

// Add Mish activation function implementation
func Mish(x *gorgonia.Node) (*gorgonia.Node, error) {
	if x == nil {
		return nil, fmt.Errorf("input node is nil")
	}

	exp, err := gorgonia.Exp(x)
	if err != nil {
		return nil, fmt.Errorf("exp error: %v", err)
	}

	added, err := gorgonia.Add(exp, gorgonia.NewConstant(1.0))
	if err != nil {
		return nil, fmt.Errorf("add error: %v", err)
	}

	softplus, err := gorgonia.Log(added)
	if err != nil {
		return nil, fmt.Errorf("log error: %v", err)
	}

	tanh, err := gorgonia.Tanh(softplus)
	if err != nil {
		return nil, fmt.Errorf("tanh error: %v", err)
	}

	result, err := gorgonia.Mul(x, tanh)
	if err != nil {
		return nil, fmt.Errorf("mul error: %v", err)
	}

	return result, nil
}
