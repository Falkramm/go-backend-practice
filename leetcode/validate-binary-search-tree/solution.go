package validatebst

/**
 * Definition for a binary tree node.
 * type TreeNode struct {
 *     Val int
 *     Left *TreeNode
 *     Right *TreeNode
 * }
 */

import "math"

func isValidSubBST(root *TreeNode, minV, maxV int) bool {
	if root == nil {
		return true
	}

	if root.Val <= minV || root.Val >= maxV {
		return false
	}

	return isValidSubBST(root.Left, minV, min(root.Val, maxV)) && isValidSubBST(root.Right, max(minV, root.Val), maxV)
}

func isValidBST(root *TreeNode) bool {
	if root == nil {
		return true
	}

	return isValidSubBST(root.Left, math.MinInt, root.Val) && isValidSubBST(root.Right, root.Val, math.MaxInt)
}
