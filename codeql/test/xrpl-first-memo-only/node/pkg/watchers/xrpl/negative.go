package xrpl

func ParseCoreFirstMemoOnly(tx Transaction) bool {
	if len(tx.Memos) == 0 {
		return false
	}
	memo := tx.Memos[0]
	if memo.MemoFormat != coreMemoFormat {
		return false
	}
	return len(memo.MemoData) > 0
}

func ParseNttFirstMemoOnly(tx Transaction) bool {
	if len(tx.Memos) == 0 {
		return false
	}
	memo := tx.Memos[0]
	return memo.MemoFormat == nttMemoFormat
}

func ParseCoreNoFallbackAfterMalformedFirst(tx Transaction) bool {
	if len(tx.Memos) == 0 {
		return false
	}
	memo := tx.Memos[0]
	if memo.MemoFormat != coreMemoFormat {
		return false
	}
	if len(memo.MemoData) < 4 {
		return false
	}
	return true
}
