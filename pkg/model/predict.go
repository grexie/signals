package model

import (
	"fmt"

	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

// Updated prediction function to match the deeper network
// Predict completes the forward pass for inference
func Predict(weights []tensor.Tensor, input []float64) ([]float64, error) {
	g := gorgonia.NewGraph()
	inputSize := len(input)

	// Input tensor with explicit shape (1 x inputSize)
	xVal := tensor.New(
		tensor.WithShape(1, inputSize),
		tensor.Of(tensor.Float64),
		tensor.WithBacking(input),
	)

	xTensor := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(1, inputSize),
		gorgonia.WithValue(xVal))

	// Load weights with explicit shapes
	w0 := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(weights[0].Shape()...),
		gorgonia.WithValue(weights[0]))
	w1 := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(weights[1].Shape()...),
		gorgonia.WithValue(weights[1]))
	w2 := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(weights[2].Shape()...),
		gorgonia.WithValue(weights[2]))
	w3 := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(weights[3].Shape()...),
		gorgonia.WithValue(weights[3]))

	// Forward pass matching training
	l0 := gorgonia.Must(gorgonia.Mul(xTensor, w0))
	l0Act := gorgonia.Must(gorgonia.Rectify(l0))

	l1 := gorgonia.Must(gorgonia.Mul(l0Act, w1))
	l1Act := gorgonia.Must(gorgonia.Rectify(l1))

	l2 := gorgonia.Must(gorgonia.Mul(l1Act, w2))
	l2Act := gorgonia.Must(gorgonia.Rectify(l2))

	pred := gorgonia.Must(gorgonia.Mul(l2Act, w3))
	predSoftmax := gorgonia.Must(gorgonia.SoftMax(pred))

	vm := gorgonia.NewTapeMachine(g)
	defer vm.Close()

	if err := vm.RunAll(); err != nil {
		return nil, fmt.Errorf("forward pass failed: %v", err)
	}

	output := predSoftmax.Value().Data().([]float64)
	return output, nil
}
