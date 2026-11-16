# LeetCode на Go

23 решения Даниила Матуса из учебной подготовки. Каждая реализация восстановлена из собственной Go-отправки со статусом **Accepted**. Код и результаты отправок проверены через интерфейс LeetCode 16 ноября 2026 года.

В подборке есть массивы и строки, стеки, списки, LRU и LFU, деревья, графы, `container/heap`, перебор и динамическое программирование.

## Решения

| № | Задача и исходный код | Тема | Условие | Отправка | Дата | Тесты LeetCode |
| --- | --- | --- | --- | --- | --- | --- |
| 49 | [Group Anagrams](group-anagrams/solution.go) | Хеширование | [LeetCode](https://leetcode.com/problems/group-anagrams/) | [Accepted](https://leetcode.com/problems/group-anagrams/submissions/2191346057/) | 2026-11-07 | 130/130 |
| 238 | [Product of Array Except Self](product-of-array-except-self/solution.go) | Префиксы и суффиксы | [LeetCode](https://leetcode.com/problems/product-of-array-except-self/) | [Accepted](https://leetcode.com/problems/product-of-array-except-self/submissions/2191355618/) | 2026-11-07 | 24/24 |
| 3 | [Longest Substring Without Repeating Characters](longest-substring-without-repeating-characters/solution.go) | Скользящее окно | [LeetCode](https://leetcode.com/problems/longest-substring-without-repeating-characters/) | [Accepted](https://leetcode.com/problems/longest-substring-without-repeating-characters/submissions/2191372328/) | 2026-11-07 | 1036/1036 |
| 15 | [3Sum](3sum/solution.go) | Хеширование, сортировка | [LeetCode](https://leetcode.com/problems/3sum/) | [Accepted](https://leetcode.com/problems/3sum/submissions/2191389951/) | 2026-11-07 | 316/316 |
| 33 | [Search in Rotated Sorted Array](search-in-rotated-sorted-array/solution.go) | Двоичный поиск | [LeetCode](https://leetcode.com/problems/search-in-rotated-sorted-array/) | [Accepted](https://leetcode.com/problems/search-in-rotated-sorted-array/submissions/2191856515/) | 2026-11-08 | 197/197 |
| 56 | [Merge Intervals](merge-intervals/solution.go) | Сортировка, интервалы | [LeetCode](https://leetcode.com/problems/merge-intervals/) | [Accepted](https://leetcode.com/problems/merge-intervals/submissions/2191417971/) | 2026-11-07 | 172/172 |
| 155 | [Min Stack](min-stack/solution.go) | Стек | [LeetCode](https://leetcode.com/problems/min-stack/) | [Accepted](https://leetcode.com/problems/min-stack/submissions/2191401785/) | 2026-11-07 | 46/46 |
| 739 | [Daily Temperatures](daily-temperatures/solution.go) | Монотонный стек | [LeetCode](https://leetcode.com/problems/daily-temperatures/) | [Accepted](https://leetcode.com/problems/daily-temperatures/submissions/2191409213/) | 2026-11-07 | 48/48 |
| 2 | [Add Two Numbers](add-two-numbers/solution.go) | Связный список | [LeetCode](https://leetcode.com/problems/add-two-numbers/) | [Accepted](https://leetcode.com/problems/add-two-numbers/submissions/2191539278/) | 2026-11-07 | 1569/1569 |
| 19 | [Remove Nth Node From End of List](remove-nth-node-from-end-of-list/solution.go) | Два указателя | [LeetCode](https://leetcode.com/problems/remove-nth-node-from-end-of-list/) | [Accepted](https://leetcode.com/problems/remove-nth-node-from-end-of-list/submissions/2191860488/) | 2026-11-08 | 208/208 |
| 146 | [LRU Cache](lru-cache/solution.go) | Хеш-таблица, container/list | [LeetCode](https://leetcode.com/problems/lru-cache/) | [Accepted](https://leetcode.com/problems/lru-cache/submissions/2191886062/) | 2026-11-08 | 23/23 |
| 460 | [LFU Cache](lfu-cache/solution.go) | Частотные списки, container/list | [LeetCode](https://leetcode.com/problems/lfu-cache/) | [Accepted](https://leetcode.com/problems/lfu-cache/submissions/2191900526/) | 2026-11-08 | 26/26 |
| 102 | [Binary Tree Level Order Traversal](binary-tree-level-order-traversal/solution.go) | Дерево, DFS с глубиной | [LeetCode](https://leetcode.com/problems/binary-tree-level-order-traversal/) | [Accepted](https://leetcode.com/problems/binary-tree-level-order-traversal/submissions/2191534201/) | 2026-11-07 | 35/35 |
| 98 | [Validate Binary Search Tree](validate-binary-search-tree/solution.go) | Дерево поиска, границы | [LeetCode](https://leetcode.com/problems/validate-binary-search-tree/) | [Accepted](https://leetcode.com/problems/validate-binary-search-tree/submissions/2191428028/) | 2026-11-07 | 86/86 |
| 200 | [Number of Islands](number-of-islands/solution.go) | Обход сетки | [LeetCode](https://leetcode.com/problems/number-of-islands/) | [Accepted](https://leetcode.com/problems/number-of-islands/submissions/2191546139/) | 2026-11-07 | 49/49 |
| 207 | [Course Schedule](course-schedule/solution.go) | Граф, поиск циклов | [LeetCode](https://leetcode.com/problems/course-schedule/) | [Accepted](https://leetcode.com/problems/course-schedule/submissions/2191551102/) | 2026-11-07 | 55/55 |
| 215 | [Kth Largest Element in an Array](kth-largest-element-in-an-array/solution.go) | container/heap | [LeetCode](https://leetcode.com/problems/kth-largest-element-in-an-array/) | [Accepted](https://leetcode.com/problems/kth-largest-element-in-an-array/submissions/2191530364/) | 2026-11-07 | 47/47 |
| 347 | [Top K Frequent Elements](top-k-frequent-elements/solution.go) | Частоты, корзины | [LeetCode](https://leetcode.com/problems/top-k-frequent-elements/) | [Accepted](https://leetcode.com/problems/top-k-frequent-elements/submissions/2191525120/) | 2026-11-07 | 26/26 |
| 1046 | [Last Stone Weight](last-stone-weight/solution.go) | container/heap | [LeetCode](https://leetcode.com/problems/last-stone-weight/) | [Accepted](https://leetcode.com/problems/last-stone-weight/submissions/2191877805/) | 2026-11-08 | 74/74 |
| 78 | [Subsets](subsets/solution.go) | Битовые маски | [LeetCode](https://leetcode.com/problems/subsets/) | [Accepted](https://leetcode.com/problems/subsets/submissions/2191552927/) | 2026-11-07 | 10/10 |
| 39 | [Combination Sum](combination-sum/solution.go) | Перебор с возвратом | [LeetCode](https://leetcode.com/problems/combination-sum/) | [Accepted](https://leetcode.com/problems/combination-sum/submissions/2191866482/) | 2026-11-08 | 160/160 |
| 198 | [House Robber](house-robber/solution.go) | Динамическое программирование | [LeetCode](https://leetcode.com/problems/house-robber/) | [Accepted](https://leetcode.com/problems/house-robber/submissions/2191520770/) | 2026-11-07 | 70/70 |
| 322 | [Coin Change](coin-change/solution.go) | Динамическое программирование | [LeetCode](https://leetcode.com/problems/coin-change/) | [Accepted](https://leetcode.com/problems/coin-change/submissions/2191486766/) | 2026-11-07 | 191/191 |

## Самостоятельная сборка

Требуется Go 1.24 или новее. Каждая задача находится в отдельном пакете, чтобы одноимённые `Constructor`, `dfs` и вспомогательные типы могли сосуществовать.

Из корня репозитория выполните команды.

```sh
cd leetcode
go test -run '^$' -timeout=30s ./...
go vet ./...
```

При подготовке архива обе команды прошли для всех 23 пакетов. Локальных тестов задач в этом модуле пока нет. Числа в таблице относятся к исходным запускам на LeetCode 7–8 ноября 2026 года.

## Подготовка исходников

Код перенесён из принятых отправок с сохранением реализаций и учебных комментариев. Для сборки вне LeetCode добавлены объявления `package`, а в задачах со списками и деревьями — отдельные файлы `types.go` с предоставляемыми площадкой типами `ListNode` и `TreeNode`. В `kth-largest-element-in-an-array` добавлен импорт `fmt` для уже присутствующего в отправке `fmt.Println`. Ко всем Go-файлам применён `gofmt`.

Функции и методы сохраняют контракты соответствующих задач LeetCode. В решениях сохранён учебный отладочный вывод, например печать кучи в задаче 215.

Дополнительное упражнение Kth Largest Element in a Stream при проверке показывало пустую историю отправок и шаблон редактора. Его условие сохранено в [общем каталоге практики](../docs/leetcode-catalog.md).

[Вернуться к коллекции](../README.md)
