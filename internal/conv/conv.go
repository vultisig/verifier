package conv

import (
	"fmt"
	"strconv"

	"github.com/labstack/echo/v4"
)

func ValueOrDefault[T comparable](value T, defaultValue T) T {
	var zero T
	if value == zero {
		return defaultValue
	}
	return value
}

func PageParamsFromCtx(c echo.Context, defaultSkip, defaultTake uint32) (uint32, uint32, error) {
	skipStr := ValueOrDefault(c.QueryParam("skip"), strconv.FormatUint(uint64(defaultSkip), 10))
	skip, err := strconv.ParseUint(skipStr, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("strconv.ParseUint(%s): %w", skipStr, err)
	}

	takeStr := ValueOrDefault(c.QueryParam("take"), strconv.FormatUint(uint64(defaultTake), 10))
	take, err := strconv.ParseUint(takeStr, 10, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("strconv.ParseUint(%s): %w", takeStr, err)
	}
	if take > 100 {
		return 0, 0, fmt.Errorf("'take' cannot be greater than 100")
	}

	return uint32(skip), uint32(take), nil
}
