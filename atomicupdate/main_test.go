package main

import (
	"fmt"
	"testing"
)

func Test_formatting(t *testing.T) {
	fmt.Printf("@@@ %1.1e\n", 1314324.436456)

	fmt.Printf("@@@ %.0f\n", approxFloat3(453124.2354))
}
