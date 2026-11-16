package coinchange

func coinChange(coins []int, amount int) int {
	if amount == 0 {
		return 0
	}
	dp := make([]int, amount+1)
	for i := range amount {
		if dp[i] == 0 && i != 0 {
			continue
		}

		for _, v := range coins {
			if i+v > amount {
				continue
			}
			if dp[i+v] == 0 {
				dp[i+v] = dp[i] + 1
			} else {
				dp[i+v] = min(dp[i+v], dp[i]+1)
			}
		}
	}
	if dp[amount] == 0 {
		return -1
	}
	return dp[amount]
}
