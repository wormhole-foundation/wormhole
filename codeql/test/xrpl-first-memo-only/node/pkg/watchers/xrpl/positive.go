package xrpl

const coreMemoFormat = "application/x-wormhole-publish"
const nttMemoFormat = "application/x-ntt-transfer"

type Memo struct {
	MemoFormat string
	MemoData   []byte
}

type Transaction struct {
	Memos []Memo
}

type FlatTransaction map[string]any

func ParseCoreWithRange(tx Transaction) bool {
	memos := tx.Memos
	for _, memo := range memos {
		if memo.MemoFormat == coreMemoFormat {
			return true
		}
	}
	return false
}

func ParseNttWithDynamicIndex(tx Transaction, i int) bool {
	if len(tx.Memos) == 0 {
		return false
	}
	memos := tx.Memos
	memo := memos[i]
	return memo.MemoFormat == nttMemoFormat
}

func ParseCoreViaScanningHelper(tx Transaction) bool {
	memo, ok := findMemoWithFormat(tx.Memos, coreMemoFormat)
	return ok && memo.MemoFormat == coreMemoFormat
}

func findMemoWithFormat(memos []Memo, format string) (Memo, bool) {
	for _, memo := range memos {
		if memo.MemoFormat == format {
			return memo, true
		}
	}
	return Memo{}, false
}

func ParseNttLastMemo(tx Transaction) bool {
	if len(tx.Memos) == 0 {
		return false
	}
	memos := tx.Memos
	memo := memos[len(memos)-1]
	return memo.MemoFormat == nttMemoFormat
}

func ParseCoreWithDirectFieldRange(tx Transaction) bool {
	for _, memo := range tx.Memos {
		if memo.MemoFormat == coreMemoFormat {
			return true
		}
	}
	return false
}

func ParseCoreWithRenamedAlias(tx Transaction) bool {
	memoList := tx.Memos
	for _, memo := range memoList {
		if memo.MemoFormat == coreMemoFormat {
			return true
		}
	}
	return false
}

func ParseNttViaFormatHelper(tx Transaction) bool {
	_, ok := findMemoWithFormat(tx.Memos, nttMemoFormat)
	return ok
}

func ParseCoreFlatMapRange(tx FlatTransaction) bool {
	memosRaw := tx["Memos"]
	memos, ok := memosRaw.([]Memo)
	if !ok {
		return false
	}
	for _, memo := range memos {
		if memo.MemoFormat == coreMemoFormat {
			return true
		}
	}
	return false
}

func ParseCoreViaScanningWrapper(tx Transaction) bool {
	return scanMemoWrapper(tx.Memos, coreMemoFormat)
}

func scanMemoWrapper(memos []Memo, format string) bool {
	_, ok := findMemoWithFormat(memos, format)
	return ok
}
