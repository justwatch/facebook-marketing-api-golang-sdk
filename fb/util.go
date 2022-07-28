package fb

import "fmt"

func truncateString(str string, n int) string {
	res := str
	if n > 0 && len(str) > n {
		res = str[0:n] + fmt.Sprintf("...[truncated_after_%d_bytes]", n)
	}

	return res
}
