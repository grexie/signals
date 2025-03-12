package model

import (
	"fmt"

	"gorgonia.org/gorgonia"
)

func CategoricalCrossEntropy(pred, target *gorgonia.Node) (*gorgonia.Node, error) {
	eps := 1e-7

	safePred, err := gorgonia.Add(pred, gorgonia.NewConstant(eps))
	if err != nil {
		return nil, fmt.Errorf("failed to add epsilon: %v", err)
	}

	logPred, err := gorgonia.Log(safePred)
	if err != nil {
		return nil, fmt.Errorf("failed to compute log: %v", err)
	}

	losses, err := gorgonia.HadamardProd(target, logPred)
	if err != nil {
		return nil, fmt.Errorf("failed to compute hadamard product: %v", err)
	}

	sumLosses, err := gorgonia.Sum(losses)
	if err != nil {
		return nil, fmt.Errorf("failed to compute sum: %v", err)
	}

	meanLoss, err := gorgonia.Mean(sumLosses)
	if err != nil {
		return nil, fmt.Errorf("failed to compute mean: %v", err)
	}

	return gorgonia.Neg(meanLoss)
}
