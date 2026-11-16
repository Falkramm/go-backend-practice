package courseschedule

func dfs(g [][]int, colors []int, v int) bool {
	if colors[v] == 2 {
		return false
	}

	colors[v] = 1
	for _, ch := range g[v] {
		if colors[ch] == 1 {
			return true
		}

		if colors[ch] == 0 && dfs(g, colors, ch) {
			return true
		}
	}

	colors[v] = 2
	return false
}
func canFinish(numCourses int, prerequisites [][]int) bool {
	g := make([][]int, numCourses)
	for _, pair := range prerequisites {
		g[pair[0]] = append(g[pair[0]], pair[1])
	}

	colors := make([]int, numCourses)

	for v := range numCourses {
		if colors[v] == 0 && dfs(g, colors, v) {
			return false
		}
	}

	return true
}
