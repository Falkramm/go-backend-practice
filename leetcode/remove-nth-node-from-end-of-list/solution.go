package removenth

/**
 * Definition for singly-linked list.
 * type ListNode struct {
 *     Val int
 *     Next *ListNode
 * }
 */
// 2

//  1 2 3 4 5

//  1) 1 1 2
//  2) 2 1 1
//  3) 3 1 0
//  4) 4 1 -1
//  5) 5 2 -1
//  -) - 3 -1

//  1
//  1 2 3 4 5

// 1) 1 1 1
// 2) 2 1 0
// 3) 3 1 -1
// 4) 4 2 -1
// 5) 5 3 -1
// -) - 4 -1
func removeNthFromEnd(head *ListNode, n int) *ListNode {
	tail := head
	tailD := head
	for tail != nil {
		tail = tail.Next
		if n == -1 {
			tailD = tailD.Next
		} else {
			n--
		}
	}

	if n == 0 {
		prev := head
		head = head.Next
		prev.Next = nil
		return head
	}

	next := tailD.Next.Next
	tailD.Next = nil
	tailD.Next = next
	return head

}
