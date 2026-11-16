package houserobber

type point struct {
	resT  int
	resNt int
}

func rob(nums []int) int {
	dp := make([]point, len(nums))
	dp[0].resT = nums[0]
	for i := 1; i < len(nums); i++ {
		dp[i].resT = dp[i-1].resNt + nums[i]
		dp[i].resNt = max(dp[i-1].resNt, dp[i-1].resT)
	}
	return max(dp[len(nums)-1].resT, dp[len(nums)-1].resNt)
}
