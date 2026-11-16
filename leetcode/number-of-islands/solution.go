package islands

type point struct {
	x int
	y int
}

func numIslands(grid [][]byte) int {
	var res int
	for i := 0; i < len(grid); i++ {
		for j := 0; j < len(grid[i]); j++ {
			if grid[i][j] == byte('0') {
				continue
			}
			grid[i][j] = byte('0')
			res++
			stack := make([]point, 0, len(grid)*len(grid[0]))
			stack = append(stack, point{
				x: i,
				y: j,
			})

			for len(stack) > 0 {
				p := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				for dx := -1; dx <= 1; dx += 1 {
					for dy := -1; dy <= 1; dy += 1 {
						if dx*dy != 0 {
							continue
						}
						dp := point{
							x: p.x + dx,
							y: p.y + dy,
						}
						if dp.x >= 0 && dp.x < len(grid) &&
							dp.y >= 0 && dp.y < len(grid[dp.x]) &&
							grid[dp.x][dp.y] == byte('1') {
							stack = append(stack, dp)
							grid[dp.x][dp.y] = byte('0')
						}
					}
				}
			}
		}
	}

	return res
}
