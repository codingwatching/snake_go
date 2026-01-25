package game

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

// PredictRequest represents a single inference task in the queue
type PredictRequest struct {
	Input   []float64
	ResChan chan []float32
}

var (
	// The core queue for all games
	predictionQueue   = make(chan PredictRequest, 200)
	workerInitialized sync.Once
)

// ONNXModel encapsulates the session and its dedicated tensors
type ONNXModel struct {
	session *ort.AdvancedSession
	input   *ort.Tensor[float32]
	output  *ort.Tensor[float32]
}

// StartInferenceService initializes the global worker that "dumps" the queue
// This is the SINGLE point of execution for all AI Brains in the system.
func StartInferenceService(modelPath string) error {
	var err error
	workerInitialized.Do(func() {
		// 1. Init ONNX Env
		err = initORT()
		if err != nil {
			return
		}

		// 2. Start the lone worker goroutine
		go func() {
			// One model instance for the entire server
			nn, nerr := loadModelInstance(modelPath)
			if nerr != nil {
				log.Printf("CRITICAL: AI Worker failed to load model: %v\n", nerr)
				return
			}
			log.Println("🚀 Global AI Optimizer-Worker is now online (Queue-based)")

			// The "Dumping" Loop
			for req := range predictionQueue {
				// Execute inference
				logits := nn.internalPredict(req.Input)
				// Send back to the specific game session
				req.ResChan <- logits
			}
		}()
	})
	return err
}

// Predict is the client method. It pushes to the queue and waits for its turn.
// This is non-blocking for the worker, and synchronous for the calling Game loop.
func Predict(input []float64) []float32 {
	resChan := make(chan []float32, 1)
	predictionQueue <- PredictRequest{
		Input:   input,
		ResChan: resChan,
	}
	return <-resChan
}

// --- Internal Helpers ---

func loadModelInstance(modelPath string) (*ONNXModel, error) {
	inputShape := ort.NewShape(1, 6, 25, 25)
	inputData := make([]float32, 1*6*25*25)
	inputTensor, _ := ort.NewTensor(inputShape, inputData)

	outputShape := ort.NewShape(1, 4)
	outputData := make([]float32, 1*4)
	outputTensor, _ := ort.NewTensor(outputShape, outputData)

	options, _ := ort.NewSessionOptions()
	defer options.Destroy()
	options.SetIntraOpNumThreads(0) // Full CPU power for the worker

	session, err := ort.NewAdvancedSession(modelPath,
		[]string{"input"}, []string{"output"},
		[]ort.ArbitraryTensor{inputTensor}, []ort.ArbitraryTensor{outputTensor}, options)

	if err != nil {
		return nil, err
	}

	return &ONNXModel{session, inputTensor, outputTensor}, nil
}

func (m *ONNXModel) internalPredict(input []float64) []float32 {
	inputData := m.input.GetData()
	for i, v := range input {
		inputData[i] = float32(v)
	}
	_ = m.session.Run()

	// Create a copy to ensure thread safety when handing back to game loop
	res := m.output.GetData()
	copied := make([]float32, len(res))
	copy(copied, res)
	return copied
}

var ortInitialized sync.Once

func initORT() error {
	var err error
	ortInitialized.Do(func() {
		// Common paths for onnxruntime
		possiblePaths := []string{
			"/opt/homebrew/opt/onnxruntime/lib/libonnxruntime.dylib", // Apple Silicon Homebrew
			"/usr/local/opt/onnxruntime/lib/libonnxruntime.dylib",    // Intel Homebrew
			"/usr/local/lib/libonnxruntime.dylib",                    // Manual install
		}

		if runtime.GOOS == "linux" {
			possiblePaths = []string{
				"/usr/lib/libonnxruntime.so",
				"/usr/local/lib/libonnxruntime.so",
			}
		}

		var foundPath string
		for _, path := range possiblePaths {
			if _, e := os.Stat(path); e == nil {
				foundPath = path
				break
			}
		}

		if foundPath == "" {
			err = fmt.Errorf("onnxruntime library not found. Please install it (e.g., 'brew install onnxruntime' on macOS)")
			return
		}

		ort.SetSharedLibraryPath(foundPath)
		err = ort.InitializeEnvironment()
	})
	return err
}
