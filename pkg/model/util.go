package model

import (
	"fmt"

	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

// Helper function to convert boolean to float (1.0 or 0.0)
func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}

func getWeightsTensor(n *gorgonia.Node) (tensor.Tensor, error) {
	v := n.Value()
	if v == nil {
		return nil, fmt.Errorf("node has nil value")
	}
	t, ok := v.(tensor.Tensor)
	if !ok {
		return nil, fmt.Errorf("value is not a tensor")
	}
	return t, nil
}
