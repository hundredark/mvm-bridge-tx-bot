package main

import (
	"context"
	"github.com/MixinNetwork/mixin/logger"
)

func main() {
	ctx := context.Background()
	logger.SetLevel(logger.DEBUG)

	StartHTTP(ctx)
}
