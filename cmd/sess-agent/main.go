package main

import (
	"github.com/DeepakSilaych/sess/internal/agent"
	"os"
)

func main() { os.Exit(agent.Run(os.Args[1:])) }
