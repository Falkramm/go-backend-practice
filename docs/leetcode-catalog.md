# LeetCode и алгоритмы

Исходный маршрут включал 20 задач, затем две задачи на heap и два кэша. 8 ноября 2026 года автор подтвердил завершение исходных 20 задач, heap, LRU и LFU. Решения писались в редакторе LeetCode. Сохранённые локальные реализации раннего цикла перечислены ниже. При сборке репозитория через сайт восстановлены 23 исходные Accepted-отправки — весь основной маршрут, LRU, LFU и Last Stone Weight. Для 703 сохранилось только условие, сайт показал отсутствие отправок. [Открыть решения и сведения об отправках](../leetcode/README.md)

## Исходные 20 задач

| № | Тема | Задача |
| --- | --- | --- |
| 1 | Хеширование | [49. Group Anagrams](https://leetcode.com/problems/group-anagrams/) |
| 2 | Префикс и суффикс | [238. Product of Array Except Self](https://leetcode.com/problems/product-of-array-except-self/) |
| 3 | Sliding window | [3. Longest Substring Without Repeating Characters](https://leetcode.com/problems/longest-substring-without-repeating-characters/) |
| 4 | Два указателя | [15. 3Sum](https://leetcode.com/problems/3sum/) |
| 5 | Стек | [155. Min Stack](https://leetcode.com/problems/min-stack/) |
| 6 | Монотонный стек | [739. Daily Temperatures](https://leetcode.com/problems/daily-temperatures/) |
| 7 | Бинарный поиск | [33. Search in Rotated Sorted Array](https://leetcode.com/problems/search-in-rotated-sorted-array/) |
| 8 | Интервалы | [56. Merge Intervals](https://leetcode.com/problems/merge-intervals/) |
| 9 | Связный список | [19. Remove Nth Node From End of List](https://leetcode.com/problems/remove-nth-node-from-end-of-list/) |
| 10 | Связный список | [2. Add Two Numbers](https://leetcode.com/problems/add-two-numbers/) |
| 11 | BFS дерева | [102. Binary Tree Level Order Traversal](https://leetcode.com/problems/binary-tree-level-order-traversal/) |
| 12 | BST | [98. Validate Binary Search Tree](https://leetcode.com/problems/validate-binary-search-tree/) |
| 13 | Обход графа | [200. Number of Islands](https://leetcode.com/problems/number-of-islands/) |
| 14 | Топологическая сортировка | [207. Course Schedule](https://leetcode.com/problems/course-schedule/) |
| 15 | Heap | [215. Kth Largest Element in an Array](https://leetcode.com/problems/kth-largest-element-in-an-array/) |
| 16 | Частоты | [347. Top K Frequent Elements](https://leetcode.com/problems/top-k-frequent-elements/) |
| 17 | Backtracking | [78. Subsets](https://leetcode.com/problems/subsets/) |
| 18 | Backtracking | [39. Combination Sum](https://leetcode.com/problems/combination-sum/) |
| 19 | Динамическое программирование | [198. House Robber](https://leetcode.com/problems/house-robber/) |
| 20 | Динамическое программирование | [322. Coin Change](https://leetcode.com/problems/coin-change/) |

## Heap и кэши

- Min-heap ограниченного размера. [703. Kth Largest Element in a Stream](https://leetcode.com/problems/kth-largest-element-in-a-stream/)
- Max-heap с повторными извлечениями. [1046. Last Stone Weight](https://leetcode.com/problems/last-stone-weight/)
- Хеш-таблица и двусвязный список, операции за O(1). [146. LRU Cache](https://leetcode.com/problems/lru-cache/)
- Частоты и давность использования, операции за среднее O(1). [460. LFU Cache](https://leetcode.com/problems/lfu-cache/)

## Сохранённые локальные алгоритмы

Контракты этих упражнений адаптированы для локального запуска. Например, `LongestUnique` возвращает саму строку, `TwoSum` возвращает пару индексов, а LRU использует `Get(key) (int, bool)`. Все восемь наборов прошли существующие тесты с `-race` при сборке репозитория.

| Упражнение | Контракт и приём | Код и тесты |
| --- | --- | --- |
| LongestUnique | Подстрока без повторов, sliding window по рунам | [День 1](../early-practice/day1/algo.go) · [Тесты](../early-practice/day1/algo_test.go) |
| TwoSum | Два индекса с заданной суммой, хеш-таблица | [День 2](../early-practice/day2/algo.go) · [Тесты](../early-practice/day2/algo_test.go) |
| MergeIntervals | Слияние интервалов, сортировка событий | [День 3](../early-practice/day3/algo.go) · [Тесты](../early-practice/day3/algo_test.go) |
| LRU | Кэш с вытеснением, собственный двусвязный список и `Mutex` | [День 4](../early-practice/day4/lru/lru.go) · [Тесты](../early-practice/day4/lru/lru_test.go) |
| IsValid | Проверка скобок, стек | [День 5](../early-practice/day5/algo.go) · [Тесты](../early-practice/day5/algo_test.go) |
| MergeSorted | Новый отсортированный массив из двух массивов | [День 6](../early-practice/day6/algo.go) · [Тесты](../early-practice/day6/algo_test.go) |
| MaxSubarray | Максимальная сумма непрерывного подмассива | [День 7](../early-practice/day7/algo.go) · [Тесты](../early-practice/day7/algo_test.go) |
| GroupAnagrams | Группировка по таблице частот | [День 8](../early-practice/day8/algo.go) · [Тесты](../early-practice/day8/algo_test.go) |

Известные ограничения сохранённых реализаций и команды запуска описаны рядом с ранней практикой. [README](../early-practice/README.md)
