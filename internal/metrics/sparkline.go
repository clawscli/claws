package metrics

import (
	"fmt"
	"math"
)

var sparkBlocks = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

const (
	SparklineWidth    = 7
	ColumnWidth       = 13
	noDataPlaceholder = "·······"
)

func RenderSparkline(result *MetricResult) string {
	if result == nil || !result.HasData || len(result.Values) == 0 {
		return fmt.Sprintf("%s  -", noDataPlaceholder)
	}

	values := result.Values
	if len(values) > SparklineWidth {
		values = values[len(values)-SparklineWidth:]
	}

	minVal, maxVal := values[0], values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	var spark string
	valRange := maxVal - minVal
	for _, v := range values {
		idx := 0
		if valRange > 0 {
			normalized := (v - minVal) / valRange
			idx = int(math.Round(normalized * float64(len(sparkBlocks)-1)))
		}
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkBlocks) {
			idx = len(sparkBlocks) - 1
		}
		spark += string(sparkBlocks[idx])
	}

	for len(spark) < SparklineWidth {
		spark = "·" + spark
	}

	return fmt.Sprintf("%s %3.0f%%", spark, result.Latest)
}
