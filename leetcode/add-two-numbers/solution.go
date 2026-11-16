package addtwonumbers

/**
 * Definition for singly-linked list.
 * type ListNode struct {
 *     Val int
 *     Next *ListNode
 * }
 */
func addTwoNumbers(l1 *ListNode, l2 *ListNode) *ListNode {
	var add int
	head := &ListNode{}
	res := head
	for {
		if l1 != nil {
			add += l1.Val
			l1 = l1.Next
		}
		if l2 != nil {
			add += l2.Val
			l2 = l2.Next
		}
		res.Val = add % 10
		add /= 10

		if l1 != nil || l2 != nil || add > 0 {
			res.Next = &ListNode{}
			res = res.Next
		} else {
			return head
		}
	}
}
