package model

import (
	"fmt"
	"math"
	"math/rand/v2"
	"runtime"

	"github.com/jedib0t/go-pretty/v6/progress"
	"gorgonia.org/gorgonia"
	"gorgonia.org/tensor"
)

func Train(pw progress.Writer, features [][]float64, labels []float64, epochs int) ([]tensor.Tensor, error) {
	tracker := progress.Tracker{
		Message: "Training",
		Total:   int64(epochs),
		Units:   progress.UnitsDefault,
	}
	pw.AppendTracker(&tracker)
	tracker.Start()

	// Network architecture with explicit shapes
	inputSize := len(features[0])
	hiddenSize1 := 64
	hiddenSize2 := 32
	hiddenSize3 := 16
	outputSize := 3
	batchSize := 32

	// Hyperparameters
	dropoutRate := 0.3
	l2Penalty := 0.01
	validateEvery := 5
	patience := 10

	// Create validation set (10%)
	totalSamples := len(features)
	validationSize := totalSamples / 10
	trainSize := totalSamples - validationSize

	// Shuffle indices
	indices := rand.Perm(totalSamples)
	trainIndices := indices[:trainSize]
	validIndices := indices[trainSize:]

	g := gorgonia.NewGraph()

	// Input and target tensors
	xTensor := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(batchSize, inputSize),
		gorgonia.WithName("x"))

	yTensor := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(batchSize, outputSize),
		gorgonia.WithName("y"))

	// Initialize weights with explicit shapes
	w0 := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(inputSize, hiddenSize1),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
		gorgonia.WithName("w0"))

	w1 := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(hiddenSize1, hiddenSize2),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
		gorgonia.WithName("w1"))

	w2 := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(hiddenSize2, hiddenSize3),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
		gorgonia.WithName("w2"))

	w3 := gorgonia.NewMatrix(g, tensor.Float64,
		gorgonia.WithShape(hiddenSize3, outputSize),
		gorgonia.WithInit(gorgonia.GlorotN(1.0)),
		gorgonia.WithName("w3"))

	// Forward pass with dropout and batch normalization
	l0 := gorgonia.Must(gorgonia.Mul(xTensor, w0))
	l0Act := gorgonia.Must(gorgonia.Rectify(l0))
	l0Drop := gorgonia.Must(gorgonia.Dropout(l0Act, dropoutRate))

	l1 := gorgonia.Must(gorgonia.Mul(l0Drop, w1))
	l1Act := gorgonia.Must(gorgonia.Rectify(l1))
	l1Drop := gorgonia.Must(gorgonia.Dropout(l1Act, dropoutRate))

	l2 := gorgonia.Must(gorgonia.Mul(l1Drop, w2))
	l2Act := gorgonia.Must(gorgonia.Rectify(l2))
	l2Drop := gorgonia.Must(gorgonia.Dropout(l2Act, dropoutRate))

	pred := gorgonia.Must(gorgonia.Mul(l2Drop, w3))
	predSoftmax := gorgonia.Must(gorgonia.SoftMax(pred))

	// Loss with L2 regularization
	crossEntropy := gorgonia.Must(gorgonia.Neg(
		gorgonia.Must(gorgonia.Mean(
			gorgonia.Must(gorgonia.Sum(
				gorgonia.Must(gorgonia.HadamardProd(
					yTensor,
					gorgonia.Must(gorgonia.Log(predSoftmax)))),
				1))))))

	// Calculate L2 regularization
	l2w0 := gorgonia.Must(gorgonia.Mean(gorgonia.Must(gorgonia.Square(w0))))
	l2w1 := gorgonia.Must(gorgonia.Mean(gorgonia.Must(gorgonia.Square(w1))))
	l2w2 := gorgonia.Must(gorgonia.Mean(gorgonia.Must(gorgonia.Square(w2))))
	l2w3 := gorgonia.Must(gorgonia.Mean(gorgonia.Must(gorgonia.Square(w3))))

	regularization := gorgonia.Must(gorgonia.Mul(
		gorgonia.NewConstant(l2Penalty),
		gorgonia.Must(gorgonia.Add(
			gorgonia.Must(gorgonia.Add(l2w0, l2w1)),
			gorgonia.Must(gorgonia.Add(l2w2, l2w3)),
		)),
	))

	loss := gorgonia.Must(gorgonia.Add(crossEntropy, regularization))

	// Calculate gradients
	if _, err := gorgonia.Grad(loss, w0, w1, w2, w3); err != nil {
		return nil, fmt.Errorf("failed to compute gradients: %v", err)
	}

	// Create VM
	vm := gorgonia.NewTapeMachine(g,
		gorgonia.WithLogger(nil),
		gorgonia.WithValueFmt("%3.3f"),
	)
	defer vm.Close()

	// Configure solver
	solver := gorgonia.NewAdamSolver(
		gorgonia.WithLearnRate(0.0001),
		gorgonia.WithBeta1(0.9),
		gorgonia.WithBeta2(0.999),
		gorgonia.WithEps(1e-8),
		gorgonia.WithClip(1.0),
	)

	// Training loop with early stopping
	bestLoss := math.Inf(1)
	noImprovementCount := 0
	bestWeights := make([]tensor.Tensor, 4)

	for epoch := range epochs {
		tracker.SetValue(int64(epoch))

		// Training phase
		trainLoss := 0.0
		batches := trainSize / batchSize

		for batch := 0; batch < batches; batch++ {
			start := batch * batchSize
			end := start + batchSize
			if end > trainSize {
				break
			}

			batchIndices := trainIndices[start:end]
			batchFeatures := tensor.New(
				tensor.WithShape(batchSize, inputSize),
				tensor.WithBacking(flattenBatchFeatures(features, batchIndices)))
			batchLabels := tensor.New(
				tensor.WithShape(batchSize, outputSize),
				tensor.WithBacking(flattenBatchLabels(labels, batchIndices, outputSize)))

			if err := gorgonia.Let(xTensor, batchFeatures); err != nil {
				return nil, fmt.Errorf("failed to update x tensor: %v", err)
			}
			if err := gorgonia.Let(yTensor, batchLabels); err != nil {
				return nil, fmt.Errorf("failed to update y tensor: %v", err)
			}

			vm.Reset()
			if err := vm.RunAll(); err != nil {
				return nil, fmt.Errorf("forward/backward pass failed: %v", err)
			}

			solver.Step(gorgonia.NodesToValueGrads(gorgonia.Nodes{w0, w1, w2, w3}))
			trainLoss += loss.Value().Data().(float64)
		}

		avgTrainLoss := trainLoss / float64(batches)

		// Validation phase
		if epoch%validateEvery == 0 {
			validLoss := 0.0
			validBatches := validationSize / batchSize

			for batch := 0; batch < validBatches; batch++ {
				start := batch * batchSize
				end := start + batchSize
				if end > validationSize {
					break
				}

				batchIndices := validIndices[start:end]
				batchFeatures := tensor.New(
					tensor.WithShape(batchSize, inputSize),
					tensor.WithBacking(flattenBatchFeatures(features, batchIndices)))
				batchLabels := tensor.New(
					tensor.WithShape(batchSize, outputSize),
					tensor.WithBacking(flattenBatchLabels(labels, batchIndices, outputSize)))

				if err := gorgonia.Let(xTensor, batchFeatures); err != nil {
					return nil, fmt.Errorf("failed to update validation x tensor: %v", err)
				}
				if err := gorgonia.Let(yTensor, batchLabels); err != nil {
					return nil, fmt.Errorf("failed to update validation y tensor: %v", err)
				}

				vm.Reset()
				if err := vm.RunAll(); err != nil {
					return nil, fmt.Errorf("validation forward pass failed: %v", err)
				}

				validLoss += loss.Value().Data().(float64)
			}

			avgValidLoss := validLoss / float64(validBatches)

			// Early stopping check
			if avgValidLoss < bestLoss {
				bestLoss = avgValidLoss
				noImprovementCount = 0
				// Save best weights
				bestWeights[0] = w0.Value().(tensor.Tensor).Clone().(tensor.Tensor)
				bestWeights[1] = w1.Value().(tensor.Tensor).Clone().(tensor.Tensor)
				bestWeights[2] = w2.Value().(tensor.Tensor).Clone().(tensor.Tensor)
				bestWeights[3] = w3.Value().(tensor.Tensor).Clone().(tensor.Tensor)
			} else {
				noImprovementCount++
			}

			tracker.Message = fmt.Sprintf("Training - TL: %.6f, VL: %.6f", avgTrainLoss, avgValidLoss)

			if noImprovementCount >= patience {
				break
			}
		}

		// Force GC every few epochs
		if epoch%5 == 0 {
			runtime.GC()
		}
	}

	tracker.MarkAsDone()

	return bestWeights, nil
}
