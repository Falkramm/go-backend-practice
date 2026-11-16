package levelorder

/**
 * Definition for a binary tree node.
 * type TreeNode struct {
 *     Val int
 *     Left *TreeNode
 *     Right *TreeNode
 * }
 */

func dfs(root *TreeNode, level int, res *[][]int) {
	if root == nil {
		return
	}
	if len(*res) <= level {
		*res = append(*res, []int{root.Val})
	} else {
		(*res)[level] = append((*res)[level], root.Val)
	}
	dfs(root.Left, level+1, res)
	dfs(root.Right, level+1, res)
}
func levelOrder(root *TreeNode) [][]int {
	res := [][]int{}
	dfs(root, 0, &res)
	return res
}
