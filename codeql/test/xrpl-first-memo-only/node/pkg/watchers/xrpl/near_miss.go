package xrpl

const unrelatedMemoFormat = "application/x-not-wormhole"

type XackMessage struct {
	Memos []Memo
}

type XackMap map[string]any

type XtcfMap map[string]any

func ParseUnrelatedMemoScanner(tx Transaction) bool {
	for _, memo := range tx.Memos {
		if memo.MemoFormat == unrelatedMemoFormat {
			return true
		}
	}
	return false
}

func ParseCoreSelectedMemoDataBytes(tx Transaction) bool {
	if len(tx.Memos) == 0 {
		return false
	}
	memo := tx.Memos[0]
	if memo.MemoFormat != coreMemoFormat {
		return false
	}
	for _, b := range memo.MemoData {
		if b == 0xff {
			return false
		}
	}
	return true
}

func ParseXackMemoScanner(msg XackMessage) bool {
	for _, memo := range msg.Memos {
		if memo.MemoFormat == coreMemoFormat {
			return true
		}
	}
	return false
}

func ParseLocalMemoArray() bool {
	memos := []Memo{{MemoFormat: coreMemoFormat}, {MemoFormat: nttMemoFormat}}
	return memos[0].MemoFormat == coreMemoFormat
}

func ParseXackRenamedAlias(msg XackMessage) bool {
	memoList := msg.Memos
	for _, memo := range memoList {
		if memo.MemoFormat == nttMemoFormat {
			return true
		}
	}
	return false
}

func ParseCoreUnrelatedHelperCalls(tx Transaction) bool {
	_, foundUnrelated := findMemoWithFormat(tx.Memos, unrelatedMemoFormat)
	selected := onlyFirstMemoHasFormat(tx.Memos, coreMemoFormat)
	return foundUnrelated && selected
}

func onlyFirstMemoHasFormat(memos []Memo, format string) bool {
	if len(memos) == 0 {
		return false
	}
	return memos[0].MemoFormat == format
}

func ParseMapShapedXack(xack XackMap) bool {
	memosRaw := xack["Memos"]
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

func ParseMapShapedXtcf(xtcf XtcfMap) bool {
	memosRaw := xtcf["Memos"]
	memos, ok := memosRaw.([]Memo)
	if !ok {
		return false
	}
	for _, memo := range memos {
		if memo.MemoFormat == nttMemoFormat {
			return true
		}
	}
	return false
}

func ParseCoreDivergentWrapperCalls(tx Transaction) bool {
	local := []Memo{{MemoFormat: coreMemoFormat}}
	return scanMemoWrapper(tx.Memos, unrelatedMemoFormat) ||
		onlyFirstMemoHasFormat(local, coreMemoFormat)
}
